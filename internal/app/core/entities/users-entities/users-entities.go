package users_entities

import (
	teams_entities "mkk_basis/rest_api/internal/app/core/entities/teams-entities"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	"time"
)

type UserRequest struct {
	Username string `json:"username" binding:"required,max=255"`
	Password string `json:"password" binding:"max=255"`
	Name     string `json:"name" binding:"required,max=255"`
}

func (u *UserRequest) ToModel(passwordHash string) *users.UserModel {
	return &users.UserModel{
		Username:     u.Username,
		Name:         u.Name,
		PasswordHash: passwordHash,
	}
}

type UserResponse struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TeamTopTaskCreatorResponse struct {
	User      UserResponse                `json:"user"`
	Team      teams_entities.TeamResponse `json:"team"`
	TaskCount int64                       `json:"task_count"`
}

func FromModelResponse(m *users.UserModel) *UserResponse {
	if m == nil {
		return nil
	}
	return &UserResponse{
		ID:        m.ID,
		Username:  m.Username,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromTeamTopTaskCreatorModel(m *users.TeamTopTaskCreatorModel) *TeamTopTaskCreatorResponse {
	if m == nil {
		return nil
	}
	return &TeamTopTaskCreatorResponse{
		User: UserResponse{
			ID:        m.UserID,
			Username:  m.Username,
			Name:      m.Name,
			CreatedAt: m.UserCreatedAt,
			UpdatedAt: m.UserUpdatedAt,
		},
		Team: teams_entities.TeamResponse{
			ID:        m.TeamID,
			Name:      m.TeamName,
			CreatedBy: m.TeamCreatedBy,
			Role:      m.TeamRole,
			JoinedAt:  timeOrZero(m.TeamJoinedAt),
			CreatedAt: m.TeamCreatedAt,
			UpdatedAt: m.TeamUpdatedAt,
		},
		TaskCount: m.TaskCount,
	}
}

func timeOrZero(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
