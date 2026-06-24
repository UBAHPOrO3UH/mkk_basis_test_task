package users_service_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	users_filter "mkk_basis/rest_api/internal/app/core/entities/users-filter"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	users_service "mkk_basis/rest_api/internal/app/core/services/users-service"
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

func TestCreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)

		repo.On("FindByUsername", "ivan", mock.Anything).
			Return((*users.UserModel)(nil), nil).
			Once()
		repo.On("Create", mock.MatchedBy(func(model *users.UserModel) bool {
			return model.Username == "ivan" &&
				model.Name == "Ivan" &&
				model.PasswordHash != "" &&
				model.PasswordHash != "password"
		}), mock.Anything).
			Return(&users.UserModel{ID: 1, Username: "ivan", Name: "Ivan"}, nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		result, err := service.CreateUser(context.Background(), &users_entities.UserRequest{
			Username: "  ivan  ",
			Password: "password",
			Name:     "Ivan",
		})

		require.NoError(t, err)
		assert.Equal(t, uint64(1), result.ID)
		assert.Equal(t, "ivan", result.Username)
	})

	t.Run("duplicate", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)

		repo.On("FindByUsername", "ivan", mock.Anything).
			Return(&users.UserModel{ID: 1, Username: "ivan"}, nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		_, err := service.CreateUser(context.Background(), &users_entities.UserRequest{
			Username: "ivan",
			Password: "password",
		})

		assert.ErrorIs(t, err, users_service.ErrUserAlreadyExists)
	})
}

func TestUpdateAndDeleteUser(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)

		repo.On("FindByID", uint64(1), mock.Anything).
			Return(&users.UserModel{ID: 1, Username: "old", Name: "Old", PasswordHash: "hash"}, nil).
			Once()
		repo.On("Update", uint64(1), mock.MatchedBy(func(model *users.UserModel) bool {
			return model.Username == "new" && model.Name == "New" && model.PasswordHash == "hash"
		}), mock.Anything).
			Return(&users.UserModel{ID: 1, Username: "new", Name: "New"}, nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		result, err := service.UpdateUser(context.Background(), 1, &users_entities.UserRequest{
			Username: "new",
			Name:     "New",
		})

		require.NoError(t, err)
		assert.Equal(t, "new", result.Username)
	})

	t.Run("delete", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		id := uint64(1)

		repo.On("FindByID", id, mock.Anything).
			Return(&users.UserModel{ID: id}, nil).
			Once()
		repo.On("Delete", id, mock.Anything).Return(&id, nil).Once()

		service := users_service.NewUserService(tm, repo)
		deletedID, err := service.DeleteUser(context.Background(), id)

		require.NoError(t, err)
		assert.Equal(t, id, *deletedID)
	})

	t.Run("not found", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByID", uint64(99), mock.Anything).
			Return((*users.UserModel)(nil), nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		_, err := service.DeleteUser(context.Background(), 99)

		assert.ErrorIs(t, err, users_service.ErrUserNotFound)
	})
}

func TestUserQueries(t *testing.T) {
	t.Run("by id", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByID", uint64(1), mock.Anything).
			Return(&users.UserModel{ID: 1, Username: "ivan"}, nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		result, err := service.GetUserByID(context.Background(), 1)

		require.NoError(t, err)
		assert.Equal(t, "ivan", result.Username)
	})

	t.Run("by username trims input", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByUsername", "ivan", mock.Anything).
			Return(&users.UserModel{ID: 1, Username: "ivan"}, nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		result, err := service.GetUserByUsername(context.Background(), " ivan ")

		require.NoError(t, err)
		assert.Equal(t, uint64(1), result.ID)
	})

	t.Run("all users", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindAll", mock.Anything).
			Return([]*users.UserModel{{ID: 1}, {ID: 2}}, nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		result, err := service.GetUsers(context.Background())

		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("filtered", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		filter := &users_filter.UserFilterRequest{Username: "iv", Limit: 10}
		expectDBRuns(tm, 1)
		repo.On("FindAllWithFilter", filter, mock.Anything).
			Return(&users.FoundUsers{
				Users:        []*users.UserModel{{ID: 1, Username: "ivan"}},
				ContentRange: 1,
			}, nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		result, err := service.GetUsersFilter(context.Background(), filter)

		require.NoError(t, err)
		assert.Equal(t, int64(1), result.ContentRange)
		assert.Equal(t, "ivan", result.Users[0].Username)
	})

	t.Run("monthly leaders", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		month := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
		expectDBRuns(tm, 1)
		repo.On("FindTopTaskCreatorsByTeamForMonth", month, mock.Anything).
			Return([]*users.TeamTopTaskCreatorModel{{
				TeamID:    10,
				TeamName:  "Core",
				UserID:    1,
				Username:  "ivan",
				TaskCount: 5,
			}}, nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		result, err := service.GetTopTaskCreatorsByTeamForMonth(context.Background(), month)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, int64(5), result[0].TaskCount)
		assert.Equal(t, "Core", result[0].Team.Name)
	})
}

func TestUpdateUserPasswordAndNotFound(t *testing.T) {
	t.Run("password changed", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByID", uint64(1), mock.Anything).
			Return(&users.UserModel{ID: 1, Username: "ivan", PasswordHash: "old hash"}, nil).
			Once()
		repo.On("Update", uint64(1), mock.MatchedBy(func(model *users.UserModel) bool {
			return model.PasswordHash != "" && model.PasswordHash != "old hash" && model.PasswordHash != "new password"
		}), mock.Anything).
			Return(&users.UserModel{ID: 1, Username: "ivan"}, nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		result, err := service.UpdateUser(context.Background(), 1, &users_entities.UserRequest{
			Username: "ivan",
			Password: "new password",
		})

		require.NoError(t, err)
		assert.Equal(t, uint64(1), result.ID)
	})

	t.Run("not found", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByID", uint64(99), mock.Anything).
			Return((*users.UserModel)(nil), nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		_, err := service.UpdateUser(context.Background(), 99, &users_entities.UserRequest{})

		assert.ErrorIs(t, err, users_service.ErrUserNotFound)
	})
}

func TestUserQueryFailures(t *testing.T) {
	t.Run("id not found", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByID", uint64(99), mock.Anything).
			Return((*users.UserModel)(nil), nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		_, err := service.GetUserByID(context.Background(), 99)
		assert.ErrorIs(t, err, users_service.ErrUserNotFound)
	})

	t.Run("username not found", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindByUsername", "missing", mock.Anything).
			Return((*users.UserModel)(nil), nil).
			Once()

		service := users_service.NewUserService(tm, repo)
		_, err := service.GetUserByUsername(context.Background(), "missing")
		assert.ErrorIs(t, err, users_service.ErrUserNotFound)
	})

	expectedErr := errors.New("database unavailable")

	t.Run("all users error", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		expectDBRuns(tm, 1)
		repo.On("FindAll", mock.Anything).
			Return([]*users.UserModel(nil), expectedErr).
			Once()

		service := users_service.NewUserService(tm, repo)
		_, err := service.GetUsers(context.Background())
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("filter error", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		filter := &users_filter.UserFilterRequest{Username: "iv"}
		expectDBRuns(tm, 1)
		repo.On("FindAllWithFilter", filter, mock.Anything).
			Return((*users.FoundUsers)(nil), expectedErr).
			Once()

		service := users_service.NewUserService(tm, repo)
		_, err := service.GetUsersFilter(context.Background(), filter)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("monthly leaders error", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		repo := mocks.NewMockUserRepository(t)
		month := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
		expectDBRuns(tm, 1)
		repo.On("FindTopTaskCreatorsByTeamForMonth", month, mock.Anything).
			Return([]*users.TeamTopTaskCreatorModel(nil), expectedErr).
			Once()

		service := users_service.NewUserService(tm, repo)
		_, err := service.GetTopTaskCreatorsByTeamForMonth(context.Background(), month)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestCreateUserRequiresPassword(t *testing.T) {
	service := users_service.NewUserService(nil, nil)

	_, err := service.CreateUser(context.Background(), &users_entities.UserRequest{
		Username: "username",
		Name:     "name",
	})
	if !errors.Is(err, users_service.ErrUserPasswordRequired) {
		t.Fatalf("expected users_service.ErrUserPasswordRequired, got %v", err)
	}
}
