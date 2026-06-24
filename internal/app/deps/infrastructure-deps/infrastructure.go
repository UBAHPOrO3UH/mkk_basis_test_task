package infrastructure_deps

import (
	"context"
	database_service "mkk_basis/rest_api/internal/app/infrastructure/database-service"
	redis_service "mkk_basis/rest_api/internal/app/infrastructure/redis-service"
	"mkk_basis/rest_api/internal/config"
)

type InfrastructureDependencies struct {
	ServerContext      context.Context
	TransactionManager database_service.TransactionManager
	RedisClient        redis_service.RedisClient
}

func NewInfrastructureDependencies(ctx context.Context) (*InfrastructureDependencies, error) {
	transactionManager := database_service.NewTransactionManager()
	redisClient := redis_service.NewRedisClient(config.CurrentConfig.Redis)

	return &InfrastructureDependencies{
		ServerContext:      ctx,
		TransactionManager: transactionManager,
		RedisClient:        redisClient,
	}, nil
}
