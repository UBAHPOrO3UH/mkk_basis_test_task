package deps

import (
	"context"
	core_deps "mkk_basis/rest_api/internal/app/deps/core-deps"
	infrastructure_deps "mkk_basis/rest_api/internal/app/deps/infrastructure-deps"
)

type DependenciesContainer struct {
	Infrastructure *infrastructure_deps.InfrastructureDependencies
	Core           *core_deps.CoreDependencies
}

func NewContainer(ctx context.Context) (*DependenciesContainer, error) {
	var err error
	infrastructure, err := infrastructure_deps.NewInfrastructureDependencies(ctx)
	if err != nil {
		return nil, err
	}
	core := core_deps.NewCoreDependencies(infrastructure)

	return &DependenciesContainer{
		Infrastructure: infrastructure,
		Core:           core,
	}, nil
}

var Container *DependenciesContainer

func InitContainer(container *DependenciesContainer) {
	Container = container
}
