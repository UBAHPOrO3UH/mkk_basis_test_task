package auth_entities

import (
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	"time"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Name     string `json:"name" binding:"required,min=2,max=255"`
}

func (r *RegisterRequest) ToUserRequest() *users_entities.UserRequest {
	return &users_entities.UserRequest{
		Username: r.Username,
		Password: r.Password,
		Name:     r.Name,
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required,max=255"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type AuthResponse struct {
	User             *users_entities.UserResponse `json:"user"`
	AccessExpiresAt  time.Time                    `json:"access_expires_at"`
	RefreshExpiresAt time.Time                    `json:"refresh_expires_at"`
}
