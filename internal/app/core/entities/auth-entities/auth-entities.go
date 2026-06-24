package auth_entities

import (
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	"time"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required,max=255"`
	Password string `json:"password" binding:"required,max=255"`
}

type AuthResponse struct {
	User             *users_entities.UserResponse `json:"user"`
	AccessExpiresAt  time.Time                    `json:"access_expires_at"`
	RefreshExpiresAt time.Time                    `json:"refresh_expires_at"`
}
