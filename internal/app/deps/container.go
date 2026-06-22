package deps

import (
	"context"
	infrastructure_deps "mkk_basis/rest_api/internal/app/deps/infrastructure-deps"
)

type DependenciesContainer struct {
	Infrastructure *infrastructure_deps.InfrastructureDependencies
}

func NewContainer(ctx context.Context) (*DependenciesContainer, error) {
	var err error
	infrastructure, err := infrastructure_deps.NewInfrastructureDependencies(ctx)
	if err != nil {
		return nil, err
	}

	return &DependenciesContainer{
		Infrastructure: infrastructure,
	}, nil
}

var Container *DependenciesContainer

func InitContainer(container *DependenciesContainer) {
	Container = container
}
