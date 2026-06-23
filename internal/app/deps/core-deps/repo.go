package core_deps

import "mkk_basis/rest_api/internal/app/core/repositorys/users"

type RepositoryDependencies struct {
	UsersRepository users.UserRepository
}

func NewRepositoriesDependencies() *RepositoryDependencies {
	return &RepositoryDependencies{
		UsersRepository: users.NewUserRepository(),
	}
}
