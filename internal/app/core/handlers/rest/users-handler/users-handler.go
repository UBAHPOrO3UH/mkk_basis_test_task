package users_handler

import (
	"context"
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	users_filter "mkk_basis/rest_api/internal/app/core/entities/users-filter"
	users_service "mkk_basis/rest_api/internal/app/core/services/users-service"
	"mkk_basis/rest_api/internal/app/deps"
)

func GetUserById(
	ctx context.Context,
	id uint64,
) (*users_entities.UserResponse, error) {
	userService := deps.Container.Core.Services.UsersService

	usersLogger.Debugf("find user with id=%d", id)

	result, err := userService.GetUserByID(ctx, id)
	if err != nil {
		usersLogger.Errorf("failed to find user with id=%d: %v", id, err)
		return nil, err
	}

	usersLogger.Debugf("user with id=%d found; result=%v", id, result)

	return result, nil
}

func GetUserByEmail(
	ctx context.Context,
	email string,
) (*users_entities.UserResponse, error) {
	userService := deps.Container.Core.Services.UsersService

	usersLogger.Debugf("find user with email=%s", email)

	result, err := userService.GetUserByEmail(ctx, email)
	if err != nil {
		usersLogger.Errorf("failed to find user with email=%s: %v", email, err)
		return nil, err
	}

	usersLogger.Debugf("user with email=%s found; result=%v", email, result)

	return result, nil
}

func GetUsers(
	ctx context.Context,
) ([]*users_entities.UserResponse, error) {
	userService := deps.Container.Core.Services.UsersService

	usersLogger.Debugf("find all users")

	result, err := userService.GetUsers(ctx)
	if err != nil {
		usersLogger.Errorf("failed to find users: %v", err)
		return nil, err
	}

	usersLogger.Debugf("found users; len=%d", len(result))

	return result, nil
}

func GetUsersFilter(
	ctx context.Context,
	params *users_filter.UserFilterRequest,
) (*users_service.FoundUsersResponse, error) {
	userService := deps.Container.Core.Services.UsersService

	usersLogger.Debugf("find users with filter: %+v", params)

	result, err := userService.GetUsersFilter(ctx, params)
	if err != nil {
		usersLogger.Errorf("failed to find users with filter: %v", err)
		return nil, err
	}

	usersLogger.Debugf(
		"found users with filter; len=%d contentRange=%d",
		len(result.Users),
		result.ContentRange,
	)

	return result, nil
}

func CreateUser(
	ctx context.Context,
	userDto *users_entities.UserRequest,
) (*users_entities.UserResponse, error) {
	userService := deps.Container.Core.Services.UsersService

	usersLogger.Debugf("create user: %+v", userDto)

	result, err := userService.CreateUser(ctx, userDto)
	if err != nil {
		usersLogger.Errorf("failed to create user: %v", err)
		return nil, err
	}

	usersLogger.Debugf("user created; result=%v", result)

	return result, nil
}

func UpdateUser(
	ctx context.Context,
	id uint64,
	userDto *users_entities.UserRequest,
) (*users_entities.UserResponse, error) {
	userService := deps.Container.Core.Services.UsersService

	usersLogger.Debugf("update user id=%d: %+v", id, userDto)

	result, err := userService.UpdateUser(ctx, id, userDto)
	if err != nil {
		usersLogger.Errorf("failed to update user id=%d: %v", id, err)
		return nil, err
	}

	usersLogger.Debugf("user updated; result=%v", result)

	return result, nil
}

func DeleteUser(
	ctx context.Context,
	id uint64,
) (*uint64, error) {
	userService := deps.Container.Core.Services.UsersService

	usersLogger.Debugf("delete user with id=%d", id)

	result, err := userService.DeleteUser(ctx, id)
	if err != nil {
		usersLogger.Errorf("failed to delete user id=%d: %v", id, err)
		return nil, err
	}

	usersLogger.Debugf("user deleted; id=%d", id)

	return result, nil
}
