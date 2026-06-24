package teams_service

import (
	"context"
	"errors"
	"fmt"
	teams_entities "mkk_basis/rest_api/internal/app/core/entities/teams-entities"
	team_members "mkk_basis/rest_api/internal/app/core/repositorys/team-members"
	"mkk_basis/rest_api/internal/app/core/repositorys/teams"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	database_service "mkk_basis/rest_api/internal/app/infrastructure/database-service"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrTeamNotFound           = errors.New("team not found")
	ErrUserNotFound           = errors.New("user not found")
	ErrTeamNameRequired       = errors.New("team name is required")
	ErrInvalidTeamRole        = errors.New("invalid team role")
	ErrInsufficientPermission = errors.New("insufficient permission")
	ErrUserAlreadyTeamMember  = errors.New("user is already a team member")
)

type TeamService interface {
	CreateTeam(ctx context.Context, ownerID uint64, request *teams_entities.CreateTeamRequest) (*teams_entities.TeamResponse, error)
	GetUserTeams(ctx context.Context, userID uint64) ([]*teams_entities.TeamResponse, error)
	GetTeamStats(ctx context.Context) ([]*teams_entities.TeamStatsResponse, error)
	InviteUser(ctx context.Context, teamID, inviterID uint64, request *teams_entities.InviteUserRequest) (*teams_entities.TeamMemberResponse, error)
}

type TeamServiceImpl struct {
	tm                   database_service.TransactionManager
	teamRepository       teams.TeamRepository
	teamMemberRepository team_members.TeamMemberRepository
	userRepository       users.UserRepository
}

func NewTeamService(
	tm database_service.TransactionManager,
	teamRepository teams.TeamRepository,
	teamMemberRepository team_members.TeamMemberRepository,
	userRepository users.UserRepository,
) TeamService {
	return &TeamServiceImpl{
		tm:                   tm,
		teamRepository:       teamRepository,
		teamMemberRepository: teamMemberRepository,
		userRepository:       userRepository,
	}
}

func (s *TeamServiceImpl) CreateTeam(
	ctx context.Context,
	ownerID uint64,
	request *teams_entities.CreateTeamRequest,
) (*teams_entities.TeamResponse, error) {
	name := ""
	if request != nil {
		name = strings.TrimSpace(request.Name)
	}
	if name == "" {
		return nil, ErrTeamNameRequired
	}

	var (
		createdTeam   *teams.TeamModel
		createdMember *team_members.TeamMemberModel
	)
	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		var err error
		createdTeam, err = s.teamRepository.Create(&teams.TeamModel{
			Name:      name,
			CreatedBy: ownerID,
		}, tx)
		if err != nil {
			return err
		}

		createdMember, err = s.teamMemberRepository.Create(&team_members.TeamMemberModel{
			TeamID: createdTeam.ID,
			UserID: ownerID,
			Role:   team_members.TeamRoleOwner,
		}, tx)
		return err
	})
	if err != nil {
		teamsLogger.Errorf("failed to create team owner_id=%d: %v", ownerID, err)
		return nil, err
	}

	teamsLogger.Infof("team created id=%d owner_id=%d", createdTeam.ID, ownerID)
	return teamResponse(createdTeam, createdMember), nil
}

func (s *TeamServiceImpl) GetUserTeams(
	ctx context.Context,
	userID uint64,
) ([]*teams_entities.TeamResponse, error) {
	responses := make([]*teams_entities.TeamResponse, 0)

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		models, err := s.teamRepository.FindAllByUserID(userID, tx)
		if err != nil {
			return err
		}

		responses = make([]*teams_entities.TeamResponse, 0, len(models))
		for _, model := range models {
			responses = append(responses, teamWithRoleResponse(model))
		}
		return nil
	})
	if err != nil {
		teamsLogger.Errorf("failed to get teams user_id=%d: %v", userID, err)
		return nil, err
	}

	return responses, nil
}

func (s *TeamServiceImpl) GetTeamStats(ctx context.Context) ([]*teams_entities.TeamStatsResponse, error) {
	responses := make([]*teams_entities.TeamStatsResponse, 0)

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		models, err := s.teamRepository.FindAllWithStats(tx)
		if err != nil {
			return err
		}

		responses = make([]*teams_entities.TeamStatsResponse, 0, len(models))
		for _, model := range models {
			responses = append(responses, &teams_entities.TeamStatsResponse{
				ID:                     model.ID,
				Name:                   model.Name,
				MemberCount:            model.MemberCount,
				DoneTasksLastSevenDays: model.DoneTasksLastSevenDays,
			})
		}
		return nil
	})
	if err != nil {
		teamsLogger.Errorf("failed to get team stats: %v", err)
		return nil, err
	}

	return responses, nil
}

func (s *TeamServiceImpl) InviteUser(
	ctx context.Context,
	teamID, inviterID uint64,
	request *teams_entities.InviteUserRequest,
) (*teams_entities.TeamMemberResponse, error) {
	if request == nil || request.UserID == 0 {
		return nil, ErrUserNotFound
	}

	role, err := inviteRole(request.Role)
	if err != nil {
		return nil, err
	}

	var createdMember *team_members.TeamMemberModel
	err = s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		team, err := s.teamRepository.FindByID(teamID, tx)
		if err != nil {
			return err
		}
		if team == nil {
			return ErrTeamNotFound
		}

		inviter, err := s.teamMemberRepository.FindByTeamIDAndUserID(teamID, inviterID, tx)
		if err != nil {
			return err
		}
		if inviter == nil || (inviter.Role != team_members.TeamRoleOwner && inviter.Role != team_members.TeamRoleAdmin) {
			return ErrInsufficientPermission
		}
		if inviter.Role == team_members.TeamRoleAdmin && role == team_members.TeamRoleAdmin {
			return ErrInsufficientPermission
		}

		user, err := s.userRepository.FindByID(request.UserID, tx)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrUserNotFound
		}

		existingMember, err := s.teamMemberRepository.FindByTeamIDAndUserID(teamID, request.UserID, tx)
		if err != nil {
			return err
		}
		if existingMember != nil {
			return ErrUserAlreadyTeamMember
		}

		createdMember, err = s.teamMemberRepository.Create(&team_members.TeamMemberModel{
			TeamID: teamID,
			UserID: request.UserID,
			Role:   role,
		}, tx)
		return err
	})
	if err != nil {
		teamsLogger.Errorf("failed to invite user_id=%d to team_id=%d by user_id=%d: %v", request.UserID, teamID, inviterID, err)
		return nil, err
	}

	teamsLogger.Infof("user_id=%d invited to team_id=%d by user_id=%d", request.UserID, teamID, inviterID)
	return teamMemberResponse(createdMember), nil
}

func inviteRole(value string) (team_members.TeamRole, error) {
	switch team_members.TeamRole(strings.TrimSpace(value)) {
	case "", team_members.TeamRoleMember:
		return team_members.TeamRoleMember, nil
	case team_members.TeamRoleAdmin:
		return team_members.TeamRoleAdmin, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidTeamRole, value)
	}
}

func teamResponse(model *teams.TeamModel, member *team_members.TeamMemberModel) *teams_entities.TeamResponse {
	return &teams_entities.TeamResponse{
		ID:        model.ID,
		Name:      model.Name,
		CreatedBy: model.CreatedBy,
		Role:      string(member.Role),
		JoinedAt:  member.JoinedAt,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func teamWithRoleResponse(model *teams.TeamWithRoleModel) *teams_entities.TeamResponse {
	return &teams_entities.TeamResponse{
		ID:        model.ID,
		Name:      model.Name,
		CreatedBy: model.CreatedBy,
		Role:      string(model.Role),
		JoinedAt:  model.JoinedAt,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func teamMemberResponse(model *team_members.TeamMemberModel) *teams_entities.TeamMemberResponse {
	return &teams_entities.TeamMemberResponse{
		TeamID:   model.TeamID,
		UserID:   model.UserID,
		Role:     string(model.Role),
		JoinedAt: model.JoinedAt,
	}
}
