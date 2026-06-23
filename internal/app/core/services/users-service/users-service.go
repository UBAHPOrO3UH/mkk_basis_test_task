package users_service

import (
	"context"
	"errors"
	"fmt"
	"mkk_basis/rest_api/internal/app/common"
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	users_filter "mkk_basis/rest_api/internal/app/core/entities/users-filter"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	database_service "mkk_basis/rest_api/internal/app/infrastructure/database-service"
	"strings"

	"gorm.io/gorm"
)

type FoundUsersResponse struct {
	Users        []*users_entities.UserResponse `json:"users"`
	ContentRange int64                          `json:"content_range"`
}

var (
	ErrUserNotFound      = errors.New("User not found")
	ErrUserAlreadyExists = errors.New("User already exists")
)

type UserService interface {
	CreateUser(
		ctx context.Context,
		request *users_entities.UserRequest,
	) (*users_entities.UserResponse, error)
	UpdateUser(
		ctx context.Context,
		id uint64,
		request *users_entities.UserRequest,
	) (*users_entities.UserResponse, error)
	DeleteUser(ctx context.Context, id uint64) (*uint64, error)
	GetUserByID(ctx context.Context, id uint64) (*users_entities.UserResponse, error)
	GetUserByEmail(
		ctx context.Context,
		email string,
	) (*users_entities.UserResponse, error)
	GetUsers(ctx context.Context) ([]*users_entities.UserResponse, error)
	GetUsersFilter(
		ctx context.Context,
		params *users_filter.UserFilterRequest,
	) (*FoundUsersResponse, error)
}

type UserServiceImpl struct {
	tm             database_service.TransactionManager
	userRepository users.UserRepository
}

func NewUserService(
	tm database_service.TransactionManager,
	userRepository users.UserRepository,
) UserService {
	return &UserServiceImpl{
		tm:             tm,
		userRepository: userRepository,
	}
}

func (s *UserServiceImpl) CreateUser(
	ctx context.Context,
	request *users_entities.UserRequest,
) (*users_entities.UserResponse, error) {
	usersLogger.Infof("create user with email=%s", request.Email)

	var createdUser *users.UserModel

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		email := strings.TrimSpace(request.Email)

		foundUser, err := s.userRepository.FindByEmail(email, tx)
		if err != nil {
			return err
		}

		if foundUser != nil {
			return fmt.Errorf("%w: email=%s", ErrUserAlreadyExists, email)
		}

		passwordHash, err := common.HashPassword(request.Password)
		if err != nil {
			return err
		}

		model := request.ToModel(passwordHash)

		createdModel, err := s.userRepository.Create(model, tx)
		if err != nil {
			return err
		}

		createdUser = createdModel

		return nil
	})
	if err != nil {
		usersLogger.Errorf("failed to create user with email=%s: %v", request.Email, err)
		return nil, err
	}

	usersLogger.Infof("user created successfully id=%d", createdUser.ID)

	return users_entities.FromModelResponse(createdUser), nil
}

func (s *UserServiceImpl) UpdateUser(
	ctx context.Context,
	id uint64,
	request *users_entities.UserRequest,
) (*users_entities.UserResponse, error) {
	usersLogger.Infof("update user id=%d", id)

	var updatedUser *users.UserModel

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		foundUser, err := s.userRepository.FindByID(id, tx)
		if err != nil {
			return err
		}
		if foundUser == nil {
			return ErrUserNotFound
		}

		email := strings.TrimSpace(request.Email)

		userWithSameEmail, err := s.userRepository.FindByEmail(email, tx)
		if err != nil {
			return err
		}

		if userWithSameEmail != nil && userWithSameEmail.ID != id {
			return fmt.Errorf("%w: email=%s", ErrUserAlreadyExists, email)
		}

		passwordHash := foundUser.PasswordHash

		if request.Password != "" {
			passwordHash, err = common.HashPassword(request.Password)
			if err != nil {
				return err
			}
		}

		model := request.ToModel(passwordHash)

		updatedModel, err := s.userRepository.Update(id, model, tx)
		if err != nil {
			return err
		}

		updatedUser = updatedModel

		return nil
	})
	if err != nil {
		usersLogger.Errorf("failed to update user id=%d: %v", id, err)
		return nil, err
	}

	usersLogger.Infof("user updated successfully id=%d", id)

	return users_entities.FromModelResponse(updatedUser), nil
}

func (s *UserServiceImpl) DeleteUser(
	ctx context.Context,
	id uint64,
) (*uint64, error) {
	usersLogger.Infof("delete user id=%d", id)

	var deletedID *uint64

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		foundUser, err := s.userRepository.FindByID(id, tx)
		if err != nil {
			return err
		}

		if foundUser == nil {
			return ErrUserNotFound
		}

		result, err := s.userRepository.Delete(id, tx)
		if err != nil {
			return err
		}

		deletedID = result

		return nil
	})
	if err != nil {
		usersLogger.Errorf("failed to delete user id=%d: %v", id, err)
		return nil, err
	}

	usersLogger.Infof("user deleted successfully id=%d", id)

	return deletedID, nil
}

func (s *UserServiceImpl) GetUserByID(
	ctx context.Context,
	id uint64,
) (*users_entities.UserResponse, error) {
	usersLogger.Infof("get user by id=%d", id)

	var user *users.UserModel

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		result, err := s.userRepository.FindByID(id, tx)
		if err != nil {
			return err
		}

		user = result

		return nil
	})
	if err != nil {
		usersLogger.Errorf("failed to get user by id=%d: %v", id, err)
		return nil, err
	}
	if user == nil {
		usersLogger.Errorf("failed to get user by id=%d", id)
		return nil, ErrUserNotFound
	}

	return users_entities.FromModelResponse(user), nil
}

func (s *UserServiceImpl) GetUserByEmail(
	ctx context.Context,
	email string,
) (*users_entities.UserResponse, error) {
	usersLogger.Infof("get user by email=%s", email)

	var user *users.UserModel

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		result, err := s.userRepository.FindByEmail(strings.TrimSpace(email), tx)
		if err != nil {
			return err
		}

		user = result

		return nil
	})
	if err != nil {
		usersLogger.Errorf("failed to get user by email=%s: %v", email, err)
		return nil, err
	}
	if user == nil {
		usersLogger.Errorf("failed to get user by email=%s", email)
		return nil, ErrUserNotFound
	}

	return users_entities.FromModelResponse(user), nil
}

func (s *UserServiceImpl) GetUsers(
	ctx context.Context,
) ([]*users_entities.UserResponse, error) {
	usersLogger.Info("get all users")

	var usersModels []*users.UserModel

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		result, err := s.userRepository.FindAll(tx)
		if err != nil {
			return err
		}

		usersModels = result

		return nil
	})
	if err != nil {
		usersLogger.Errorf("failed to get users: %v", err)
		return nil, err
	}

	response := make([]*users_entities.UserResponse, 0, len(usersModels))

	for _, user := range usersModels {
		response = append(response, users_entities.FromModelResponse(user))
	}

	usersLogger.Infof("found %d users", len(response))

	return response, nil
}

func (s *UserServiceImpl) GetUsersFilter(
	ctx context.Context,
	params *users_filter.UserFilterRequest,
) (*FoundUsersResponse, error) {
	usersLogger.Info("get users with filter")

	var foundUsers *users.FoundUsers

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		result, err := s.userRepository.FindAllWithFilter(params, tx)
		if err != nil {
			return err
		}

		foundUsers = result

		return nil
	})
	if err != nil {
		usersLogger.Errorf("failed to get users with filter: %v", err)
		return nil, err
	}

	response := make([]*users_entities.UserResponse, 0, len(foundUsers.Users))

	for _, user := range foundUsers.Users {
		response = append(response, users_entities.FromModelResponse(user))
	}

	usersLogger.Infof("found %d users with filter", len(response))

	return &FoundUsersResponse{
		Users:        response,
		ContentRange: foundUsers.ContentRange,
	}, nil
}
