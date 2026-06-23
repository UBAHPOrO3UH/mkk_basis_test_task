package core_deps

import (
	users_service "mkk_basis/rest_api/internal/app/core/services/users-service"
	infrastructure_deps "mkk_basis/rest_api/internal/app/deps/infrastructure-deps"
)

type ServicesDependencies struct {
	UsersService users_service.UserService
}

func NewServicesDependencies(infrastructureDeps *infrastructure_deps.InfrastructureDependencies, repoDeps *RepositoryDependencies) *ServicesDependencies {
	usersService := users_service.NewUserService(infrastructureDeps.TransactionManager, repoDeps.UsersRepository)
	return &ServicesDependencies{
		UsersService: usersService,
	}
}
