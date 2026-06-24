package users_service

import (
	"context"
	"errors"
	"testing"

	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
)

func TestCreateUserRequiresPassword(t *testing.T) {
	service := &UserServiceImpl{}

	_, err := service.CreateUser(context.Background(), &users_entities.UserRequest{
		Username: "username",
		Name:     "name",
	})
	if !errors.Is(err, ErrUserPasswordRequired) {
		t.Fatalf("expected ErrUserPasswordRequired, got %v", err)
	}
}
