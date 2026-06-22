package infrastructure_deps

import (
	"context"
	database_service "mkk_basis/rest_api/internal/app/infrastructure/database-service"
)

type InfrastructureDependencies struct {
	ServerContext      context.Context
	TransactionManager database_service.TransactionManager
}

func NewInfrastructureDependencies(ctx context.Context) (*InfrastructureDependencies, error) {
	transactionManager := database_service.NewTransactionManager()

	return &InfrastructureDependencies{
		ServerContext:      ctx,
		TransactionManager: transactionManager,
	}, nil
}
