package core_deps

import (
	auth_service "mkk_basis/rest_api/internal/app/core/services/auth-service"
	users_service "mkk_basis/rest_api/internal/app/core/services/users-service"
	infrastructure_deps "mkk_basis/rest_api/internal/app/deps/infrastructure-deps"
	"mkk_basis/rest_api/internal/config"
)

type ServicesDependencies struct {
	UsersService users_service.UserService
	AuthService  auth_service.AuthService
}

func NewServicesDependencies(infrastructureDeps *infrastructure_deps.InfrastructureDependencies, repoDeps *RepositoryDependencies) *ServicesDependencies {
	usersService := users_service.NewUserService(infrastructureDeps.TransactionManager, repoDeps.UsersRepository)
	tokenService := auth_service.NewTokenService(config.CurrentConfig.Auth)
	authService := auth_service.NewAuthService(
		infrastructureDeps.TransactionManager, repoDeps.UsersRepository, usersService, tokenService,
	)
	return &ServicesDependencies{
		UsersService: usersService,
		AuthService:  authService,
	}
}
