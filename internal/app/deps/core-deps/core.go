package core_deps

import infrastructure_deps "mkk_basis/rest_api/internal/app/deps/infrastructure-deps"

type CoreDependencies struct {
	Repo     *RepositoryDependencies
	Services *ServicesDependencies
}

func NewCoreDependencies(infrastructure *infrastructure_deps.InfrastructureDependencies) *CoreDependencies {
	repoDeps := NewRepositoriesDependencies()
	serviceDeps := NewServicesDependencies(infrastructure, repoDeps)
	return &CoreDependencies{
		Repo:     repoDeps,
		Services: serviceDeps,
	}
}
