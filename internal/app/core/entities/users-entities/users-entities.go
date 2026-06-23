package users_entities

import (
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	"time"
)

type UserRequest struct {
	Username string `json:"username" binding:"required,max=255"`
	Password string `json:"password" binding:"required,max=255"`
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
