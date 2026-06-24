package teams_service_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	teams_entities "mkk_basis/rest_api/internal/app/core/entities/teams-entities"
	team_members "mkk_basis/rest_api/internal/app/core/repositorys/team-members"
	"mkk_basis/rest_api/internal/app/core/repositorys/teams"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	teams_service "mkk_basis/rest_api/internal/app/core/services/teams-service"
	"mkk_basis/rest_api/internal/mocks"
	"testing"
	"time"
)

func expectDBRuns(tm *mocks.MockTransactionManager, count int) {
	tm.EXPECT().
		DBRun(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context, *gorm.DB) error, _ ...*sql.TxOptions) error {
			return fn(ctx, &gorm.DB{})
		}).
		Times(count)
}

func TestCreateTeam(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		teamRepo := mocks.NewMockTeamRepository(t)
		memberRepo := mocks.NewMockTeamMemberRepository(t)
		expectDBRuns(tm, 1)

		teamRepo.On("Create", mock.MatchedBy(func(model *teams.TeamModel) bool {
			return model.Name == "Core" && model.CreatedBy == 7
		}), mock.Anything).
			Return(&teams.TeamModel{ID: 10, Name: "Core", CreatedBy: 7}, nil).
			Once()
		memberRepo.On("Create", mock.MatchedBy(func(model *team_members.TeamMemberModel) bool {
			return model.TeamID == 10 && model.UserID == 7 && model.Role == team_members.TeamRoleOwner
		}), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7, Role: team_members.TeamRoleOwner}, nil).
			Once()

		service := teams_service.NewTeamService(tm, teamRepo, memberRepo, mocks.NewMockUserRepository(t))
		result, err := service.CreateTeam(context.Background(), 7, &teams_entities.CreateTeamRequest{Name: "  Core  "})

		require.NoError(t, err)
		assert.Equal(t, uint64(10), result.ID)
		assert.Equal(t, "owner", result.Role)
	})

	t.Run("name required", func(t *testing.T) {
		service := teams_service.NewTeamService(nil, nil, nil, nil)

		_, err := service.CreateTeam(context.Background(), 7, nil)
		assert.ErrorIs(t, err, teams_service.ErrTeamNameRequired)

		_, err = service.CreateTeam(context.Background(), 7, &teams_entities.CreateTeamRequest{Name: " "})
		assert.ErrorIs(t, err, teams_service.ErrTeamNameRequired)
	})
}

func TestGetTeamsAndStats(t *testing.T) {
	t.Run("user teams", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		teamRepo := mocks.NewMockTeamRepository(t)
		expectDBRuns(tm, 1)
		joinedAt := time.Now().UTC()

		teamRepo.On("FindAllByUserID", uint64(7), mock.Anything).
			Return([]*teams.TeamWithRoleModel{{
				ID:        10,
				Name:      "Core",
				CreatedBy: 7,
				Role:      team_members.TeamRoleOwner,
				JoinedAt:  joinedAt,
			}}, nil).
			Once()

		service := teams_service.NewTeamService(
			tm,
			teamRepo,
			mocks.NewMockTeamMemberRepository(t),
			mocks.NewMockUserRepository(t),
		)
		result, err := service.GetUserTeams(context.Background(), 7)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "Core", result[0].Name)
		assert.Equal(t, "owner", result[0].Role)
	})

	t.Run("stats", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		teamRepo := mocks.NewMockTeamRepository(t)
		expectDBRuns(tm, 1)

		teamRepo.On("FindAllWithStats", mock.Anything).
			Return([]*teams.TeamStatsModel{{
				ID:                     10,
				Name:                   "Core",
				MemberCount:            3,
				DoneTasksLastSevenDays: 5,
			}}, nil).
			Once()

		service := teams_service.NewTeamService(
			tm,
			teamRepo,
			mocks.NewMockTeamMemberRepository(t),
			mocks.NewMockUserRepository(t),
		)
		result, err := service.GetTeamStats(context.Background())

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, int64(3), result[0].MemberCount)
		assert.Equal(t, int64(5), result[0].DoneTasksLastSevenDays)
	})
}

func TestInviteUser(t *testing.T) {
	t.Run("owner invites member", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		teamRepo := mocks.NewMockTeamRepository(t)
		memberRepo := mocks.NewMockTeamMemberRepository(t)
		userRepo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)

		teamRepo.On("FindByID", uint64(10), mock.Anything).
			Return(&teams.TeamModel{ID: 10, Name: "Core"}, nil).
			Once()
		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7, Role: team_members.TeamRoleOwner}, nil).
			Once()
		userRepo.On("FindByID", uint64(8), mock.Anything).
			Return(&users.UserModel{ID: 8, Username: "member"}, nil).
			Once()
		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(8), mock.Anything).
			Return((*team_members.TeamMemberModel)(nil), nil).
			Once()
		memberRepo.On("Create", mock.MatchedBy(func(model *team_members.TeamMemberModel) bool {
			return model.TeamID == 10 && model.UserID == 8 && model.Role == team_members.TeamRoleMember
		}), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 8, Role: team_members.TeamRoleMember}, nil).
			Once()

		service := teams_service.NewTeamService(tm, teamRepo, memberRepo, userRepo)
		result, err := service.InviteUser(context.Background(), 10, 7, &teams_entities.InviteUserRequest{UserID: 8})

		require.NoError(t, err)
		assert.Equal(t, "member", result.Role)
	})

	t.Run("admin cannot invite admin", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		teamRepo := mocks.NewMockTeamRepository(t)
		memberRepo := mocks.NewMockTeamMemberRepository(t)
		expectDBRuns(tm, 1)

		teamRepo.On("FindByID", uint64(10), mock.Anything).
			Return(&teams.TeamModel{ID: 10}, nil).
			Once()
		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7, Role: team_members.TeamRoleAdmin}, nil).
			Once()

		service := teams_service.NewTeamService(tm, teamRepo, memberRepo, mocks.NewMockUserRepository(t))
		_, err := service.InviteUser(context.Background(), 10, 7, &teams_entities.InviteUserRequest{
			UserID: 8,
			Role:   "admin",
		})

		assert.ErrorIs(t, err, teams_service.ErrInsufficientPermission)
	})

	t.Run("validation", func(t *testing.T) {
		service := teams_service.NewTeamService(nil, nil, nil, nil)

		_, err := service.InviteUser(context.Background(), 1, 1, nil)
		assert.ErrorIs(t, err, teams_service.ErrUserNotFound)

		_, err = service.InviteUser(context.Background(), 1, 1, &teams_entities.InviteUserRequest{
			UserID: 2,
			Role:   "owner",
		})
		assert.ErrorIs(t, err, teams_service.ErrInvalidTeamRole)
	})
}

func TestTeamQueryErrors(t *testing.T) {
	expectedErr := errors.New("database unavailable")

	t.Run("user teams", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockTeamRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindAllByUserID", uint64(7), mock.Anything).
			Return([]*teams.TeamWithRoleModel(nil), expectedErr).
			Once()

		service := teams_service.NewTeamService(tm, repo, mocks.NewMockTeamMemberRepository(t), mocks.NewMockUserRepository(t))
		_, err := service.GetUserTeams(context.Background(), 7)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("stats", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockTeamRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindAllWithStats", mock.Anything).
			Return([]*teams.TeamStatsModel(nil), expectedErr).
			Once()

		service := teams_service.NewTeamService(tm, repo, mocks.NewMockTeamMemberRepository(t), mocks.NewMockUserRepository(t))
		_, err := service.GetTeamStats(context.Background())
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestCreateTeamRepositoryError(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	repo := mocks.NewMockTeamRepository(t)
	expectedErr := errors.New("insert failed")
	expectDBRuns(tm, 1)
	repo.On("Create", mock.Anything, mock.Anything).
		Return((*teams.TeamModel)(nil), expectedErr).
		Once()

	service := teams_service.NewTeamService(tm, repo, mocks.NewMockTeamMemberRepository(t), mocks.NewMockUserRepository(t))
	_, err := service.CreateTeam(context.Background(), 7, &teams_entities.CreateTeamRequest{Name: "Core"})

	assert.ErrorIs(t, err, expectedErr)
}

func TestInviteUserFailures(t *testing.T) {
	t.Run("team not found", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockTeamRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByID", uint64(10), mock.Anything).
			Return((*teams.TeamModel)(nil), nil).
			Once()

		service := teams_service.NewTeamService(tm, repo, mocks.NewMockTeamMemberRepository(t), mocks.NewMockUserRepository(t))
		_, err := service.InviteUser(context.Background(), 10, 7, &teams_entities.InviteUserRequest{UserID: 8})
		assert.ErrorIs(t, err, teams_service.ErrTeamNotFound)
	})

	t.Run("user not found", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		teamRepo := mocks.NewMockTeamRepository(t)
		memberRepo := mocks.NewMockTeamMemberRepository(t)
		userRepo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		teamRepo.On("FindByID", uint64(10), mock.Anything).
			Return(&teams.TeamModel{ID: 10}, nil).
			Once()
		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7, Role: team_members.TeamRoleOwner}, nil).
			Once()
		userRepo.On("FindByID", uint64(8), mock.Anything).
			Return((*users.UserModel)(nil), nil).
			Once()

		service := teams_service.NewTeamService(tm, teamRepo, memberRepo, userRepo)
		_, err := service.InviteUser(context.Background(), 10, 7, &teams_entities.InviteUserRequest{UserID: 8})
		assert.ErrorIs(t, err, teams_service.ErrUserNotFound)
	})

	t.Run("already a member", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		teamRepo := mocks.NewMockTeamRepository(t)
		memberRepo := mocks.NewMockTeamMemberRepository(t)
		userRepo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		teamRepo.On("FindByID", uint64(10), mock.Anything).
			Return(&teams.TeamModel{ID: 10}, nil).
			Once()
		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7, Role: team_members.TeamRoleOwner}, nil).
			Once()
		userRepo.On("FindByID", uint64(8), mock.Anything).
			Return(&users.UserModel{ID: 8}, nil).
			Once()
		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(8), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 8}, nil).
			Once()

		service := teams_service.NewTeamService(tm, teamRepo, memberRepo, userRepo)
		_, err := service.InviteUser(context.Background(), 10, 7, &teams_entities.InviteUserRequest{UserID: 8})
		assert.ErrorIs(t, err, teams_service.ErrUserAlreadyTeamMember)
	})
}
