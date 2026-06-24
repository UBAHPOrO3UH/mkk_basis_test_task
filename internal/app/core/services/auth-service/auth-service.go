package auth_service

import (
	"context"
	"errors"
	"mkk_basis/rest_api/internal/app/common"
	auth_entities "mkk_basis/rest_api/internal/app/core/entities/auth-entities"
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	users_service "mkk_basis/rest_api/internal/app/core/services/users-service"
	database_service "mkk_basis/rest_api/internal/app/infrastructure/database-service"
	"strings"

	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

type AuthService interface {
	Register(context.Context, *users_entities.UserRequest) (*users_entities.UserResponse, error)
	Login(context.Context, *auth_entities.LoginRequest) (*users_entities.UserResponse, *TokenPair, error)
	ValidateAccess(string) (*Claims, error)
	RefreshAccess(context.Context, string) (*AccessToken, error)
}

type AuthServiceImpl struct {
	tm             database_service.TransactionManager
	userRepository users.UserRepository
	usersService   users_service.UserService
	tokenService   *TokenService
}

func NewAuthService(
	tm database_service.TransactionManager,
	userRepository users.UserRepository,
	usersService users_service.UserService,
	tokenService *TokenService,
) AuthService {
	return &AuthServiceImpl{
		tm:             tm,
		userRepository: userRepository,
		usersService:   usersService,
		tokenService:   tokenService,
	}
}

func (s *AuthServiceImpl) Register(
	ctx context.Context,
	request *users_entities.UserRequest,
) (*users_entities.UserResponse, error) {
	authLogger.Infof("register user username=%s", request.Username)

	user, err := s.usersService.CreateUser(ctx, request)
	if err != nil {
		authLogger.Errorf("failed to register user username=%s: %v", request.Username, err)
		return nil, err
	}

	authLogger.Infof("user registered id=%d username=%s", user.ID, user.Username)
	return user, nil
}

func (s *AuthServiceImpl) Login(
	ctx context.Context,
	request *auth_entities.LoginRequest,
) (*users_entities.UserResponse, *TokenPair, error) {
	username := strings.TrimSpace(request.Username)
	authLogger.Infof("authenticate user username=%s", username)

	var user *users.UserModel
	err := s.tm.DBRun(ctx, func(_ context.Context, tx *gorm.DB) error {
		var err error
		user, err = s.userRepository.FindByUsername(username, tx)
		return err
	})
	if err != nil {
		authLogger.Errorf("failed to find user username=%s: %v", username, err)
		return nil, nil, err
	}

	if user == nil || common.ComparePassword(user.PasswordHash, request.Password) != nil {
		authLogger.Warnf("invalid credentials username=%s", username)
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.tokenService.IssuePair(user.ID, user.Username)
	if err != nil {
		authLogger.Errorf("failed to issue tokens user_id=%d username=%s: %v", user.ID, user.Username, err)
		return nil, nil, err
	}

	authLogger.Infof("user authenticated user_id=%d username=%s", user.ID, user.Username)
	return users_entities.FromModelResponse(user), tokens, nil
}

func (s *AuthServiceImpl) ValidateAccess(token string) (*Claims, error) {
	claims, err := s.tokenService.ParseAccess(token)
	if err != nil {
		authLogger.Debugf("access token validation failed: %v", err)
		return nil, err
	}
	return claims, nil
}

func (s *AuthServiceImpl) RefreshAccess(ctx context.Context, token string) (*AccessToken, error) {
	claims, err := s.tokenService.ParseRefresh(token)
	if err != nil {
		authLogger.Warnf("refresh token validation failed: %v", err)
		return nil, err
	}
	userID, err := claims.UserID()
	if err != nil {
		authLogger.Warnf("refresh token has invalid subject: %v", err)
		return nil, err
	}
	authLogger.Debugf("refresh access token user_id=%d", userID)

	var user *users.UserModel
	err = s.tm.DBRun(ctx, func(_ context.Context, tx *gorm.DB) error {
		var findErr error
		user, findErr = s.userRepository.FindByID(userID, tx)
		return findErr
	})
	if err != nil {
		authLogger.Errorf("failed to find user during token refresh user_id=%d: %v", userID, err)
		return nil, err
	}
	if user == nil {
		authLogger.Warnf("token refresh rejected: user not found user_id=%d", userID)
		return nil, ErrInvalidToken
	}

	accessToken, err := s.tokenService.IssueAccess(user.ID, user.Username)
	if err != nil {
		authLogger.Errorf("failed to refresh access token user_id=%d: %v", user.ID, err)
		return nil, err
	}

	authLogger.Infof("access token refreshed user_id=%d", user.ID)
	return accessToken, nil
}
