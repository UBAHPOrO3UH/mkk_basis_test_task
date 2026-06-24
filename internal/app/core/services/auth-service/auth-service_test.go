package auth_service_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"mkk_basis/rest_api/internal/app/common"
	auth_entities "mkk_basis/rest_api/internal/app/core/entities/auth-entities"
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	auth_service "mkk_basis/rest_api/internal/app/core/services/auth-service"
	"mkk_basis/rest_api/internal/config"
	"mkk_basis/rest_api/internal/mocks"
	"strings"
	"testing"
)

func expectDBRuns(tm *mocks.MockTransactionManager, count int) {
	tm.EXPECT().
		DBRun(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context, *gorm.DB) error, _ ...*sql.TxOptions) error {
			return fn(ctx, &gorm.DB{})
		}).
		Times(count)
}

func tokenService() *auth_service.TokenService {
	return auth_service.NewTokenService(&config.AuthConfig{
		JWTSecret:             "01234567890123456789012345678901",
		JWTIssuer:             "test",
		AccessTokenTTLMinutes: 15,
		RefreshTokenTTLHours:  24,
	})
}

func TestRegister(t *testing.T) {
	userService := mocks.NewMockUserService(t)
	request := &users_entities.UserRequest{Username: "ivan", Password: "password"}
	expected := &users_entities.UserResponse{ID: 1, Username: "ivan"}
	userService.On("CreateUser", mock.Anything, request).Return(expected, nil).Once()

	service := auth_service.NewAuthService(
		mocks.NewMockTransactionManager(t),
		mocks.NewMockUserRepository(t),
		userService,
		tokenService(),
	)
	result, err := service.Register(context.Background(), request)

	require.NoError(t, err)
	assert.Same(t, expected, result)
}

func TestLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		hash, err := common.HashPassword("password")
		require.NoError(t, err)

		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByUsername", "ivan", mock.Anything).
			Return(&users.UserModel{ID: 1, Username: "ivan", PasswordHash: hash}, nil).
			Once()

		service := auth_service.NewAuthService(tm, repo, mocks.NewMockUserService(t), tokenService())
		user, tokens, err := service.Login(context.Background(), &auth_entities.LoginRequest{
			Username: " ivan ",
			Password: "password",
		})

		require.NoError(t, err)
		assert.Equal(t, uint64(1), user.ID)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByUsername", "ivan", mock.Anything).
			Return((*users.UserModel)(nil), nil).
			Once()

		service := auth_service.NewAuthService(tm, repo, mocks.NewMockUserService(t), tokenService())
		_, _, err := service.Login(context.Background(), &auth_entities.LoginRequest{
			Username: "ivan",
			Password: "wrong",
		})

		assert.ErrorIs(t, err, auth_service.ErrInvalidCredentials)
	})
}

func TestValidateAndRefreshAccess(t *testing.T) {
	tokens := tokenService()
	pair, err := tokens.IssuePair(1, "ivan")
	require.NoError(t, err)

	tm := mocks.NewMockTransactionManager(t)
	repo := mocks.NewMockUserRepository(t)
	expectDBRuns(tm, 1)
	repo.On("FindByID", uint64(1), mock.Anything).
		Return(&users.UserModel{ID: 1, Username: "ivan"}, nil).
		Once()

	service := auth_service.NewAuthService(tm, repo, mocks.NewMockUserService(t), tokens)
	claims, err := service.ValidateAccess(pair.AccessToken)
	require.NoError(t, err)
	userID, err := claims.UserID()
	require.NoError(t, err)
	assert.Equal(t, uint64(1), userID)

	access, err := service.RefreshAccess(context.Background(), pair.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, access.Token)
}

func TestAuthFailurePaths(t *testing.T) {
	expectedErr := errors.New("dependency unavailable")

	t.Run("register", func(t *testing.T) {
		userService := mocks.NewMockUserService(t)
		request := &users_entities.UserRequest{Username: "ivan", Password: "password"}
		userService.On("CreateUser", mock.Anything, request).
			Return((*users_entities.UserResponse)(nil), expectedErr).
			Once()

		service := auth_service.NewAuthService(
			mocks.NewMockTransactionManager(t),
			mocks.NewMockUserRepository(t),
			userService,
			tokenService(),
		)
		_, err := service.Register(context.Background(), request)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("login repository error", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByUsername", "ivan", mock.Anything).
			Return((*users.UserModel)(nil), expectedErr).
			Once()

		service := auth_service.NewAuthService(tm, repo, mocks.NewMockUserService(t), tokenService())
		_, _, err := service.Login(context.Background(), &auth_entities.LoginRequest{
			Username: "ivan",
			Password: "password",
		})
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("invalid access token", func(t *testing.T) {
		service := auth_service.NewAuthService(nil, nil, nil, tokenService())
		_, err := service.ValidateAccess("not-a-token")
		assert.ErrorIs(t, err, auth_service.ErrInvalidToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		service := auth_service.NewAuthService(nil, nil, nil, tokenService())
		_, err := service.RefreshAccess(context.Background(), "not-a-token")
		assert.ErrorIs(t, err, auth_service.ErrInvalidToken)
	})

	t.Run("refresh user not found", func(t *testing.T) {
		tokens := tokenService()
		pair, err := tokens.IssuePair(1, "ivan")
		require.NoError(t, err)

		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByID", uint64(1), mock.Anything).
			Return((*users.UserModel)(nil), nil).
			Once()

		service := auth_service.NewAuthService(tm, repo, mocks.NewMockUserService(t), tokens)
		_, err = service.RefreshAccess(context.Background(), pair.RefreshToken)
		assert.ErrorIs(t, err, auth_service.ErrInvalidToken)
	})
}

func TestTokenServiceRejectsInvalidInputs(t *testing.T) {
	service := tokenService()

	_, err := service.IssuePair(0, "ivan")
	assert.ErrorIs(t, err, auth_service.ErrInvalidToken)

	_, err = service.IssueAccess(1, " ")
	assert.ErrorIs(t, err, auth_service.ErrInvalidToken)

	invalidTokens := []string{
		"",
		"one.two",
		"%%%." + "e30" + ".signature",
		"e30.%%%.signature",
		"e30.e30.%%%",
	}
	for _, token := range invalidTokens {
		_, err = service.ParseAccess(token)
		assert.ErrorIs(t, err, auth_service.ErrInvalidToken)
	}
}

func TestRefreshAccessDependencyFailures(t *testing.T) {
	t.Run("repository error", func(t *testing.T) {
		tokens := tokenService()
		pair, err := tokens.IssuePair(1, "ivan")
		require.NoError(t, err)

		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		expectedErr := errors.New("database unavailable")
		repo.On("FindByID", uint64(1), mock.Anything).
			Return((*users.UserModel)(nil), expectedErr).
			Once()

		service := auth_service.NewAuthService(tm, repo, mocks.NewMockUserService(t), tokens)
		_, err = service.RefreshAccess(context.Background(), pair.RefreshToken)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("access token issue error", func(t *testing.T) {
		tokens := tokenService()
		pair, err := tokens.IssuePair(1, "ivan")
		require.NoError(t, err)

		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByID", uint64(1), mock.Anything).
			Return(&users.UserModel{ID: 1, Username: ""}, nil).
			Once()

		service := auth_service.NewAuthService(tm, repo, mocks.NewMockUserService(t), tokens)
		_, err = service.RefreshAccess(context.Background(), pair.RefreshToken)
		assert.ErrorIs(t, err, auth_service.ErrInvalidToken)
	})
}

func TestTokenServiceIssueAndParse(t *testing.T) {
	service := tokenService()

	pair, err := service.IssuePair(42, "ivan")
	require.NoError(t, err)

	accessClaims, err := service.ParseAccess(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "42", accessClaims.Subject)
	assert.Equal(t, "ivan", accessClaims.Username)

	_, err = service.ParseAccess(pair.RefreshToken)
	assert.ErrorIs(t, err, auth_service.ErrInvalidToken)
}

func TestTokenServiceRejectsTamperedToken(t *testing.T) {
	service := tokenService()
	pair, err := service.IssuePair(42, "ivan")
	require.NoError(t, err)

	parts := strings.Split(pair.RefreshToken, ".")
	require.Len(t, parts, 3)
	parts[1] = parts[1][:len(parts[1])-1] + "A"

	_, err = service.ParseRefresh(strings.Join(parts, "."))
	assert.True(t, errors.Is(err, auth_service.ErrInvalidToken))
}
