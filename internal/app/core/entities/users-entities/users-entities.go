package users_entities

import (
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	"time"
)

type UserRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Name     string `json:"name" binding:"required,min=2,max=255"`
}

func (u *UserRequest) ToModel(passwordHash string) *users.UserModel {
	return &users.UserModel{
		Email:        u.Email,
		Name:         u.Name,
		PasswordHash: passwordHash,
	}
}

type UserResponse struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
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
		Email:     m.Email,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
