package auth_handler

import (
	"context"
	auth_entities "mkk_basis/rest_api/internal/app/core/entities/auth-entities"
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	auth_service "mkk_basis/rest_api/internal/app/core/services/auth-service"
	"mkk_basis/rest_api/internal/app/deps"
)

func Register(
	ctx context.Context,
	request *auth_entities.RegisterRequest,
) (*users_entities.UserResponse, error) {
	authLogger.Infof("register user username=%s", request.Username)

	user, err := deps.Container.Core.Services.AuthService.Register(ctx, request)
	if err != nil {
		authLogger.Errorf("failed to register user username=%s: %v", request.Username, err)
		return nil, err
	}

	authLogger.Infof("user registered id=%d username=%s", user.ID, user.Username)
	return user, nil
}

func Login(
	ctx context.Context,
	request *auth_entities.LoginRequest,
) (*users_entities.UserResponse, *auth_service.TokenPair, error) {
	authLogger.Infof("login user username=%s", request.Username)

	user, tokens, err := deps.Container.Core.Services.AuthService.Login(ctx, request)
	if err != nil {
		authLogger.Errorf("failed to login user username=%s: %v", request.Username, err)
		return nil, nil, err
	}

	authLogger.Infof("user logged in id=%d username=%s", user.ID, user.Username)
	return user, tokens, nil
}
