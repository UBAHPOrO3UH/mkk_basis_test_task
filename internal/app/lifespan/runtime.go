package lifespan

import (
	"context"
	"fmt"
	"mkk_basis/rest_api/internal/app/deps"
)

func PreRun(ctx context.Context, reload bool) error {
	container, err := deps.NewContainer(ctx)
	if err != nil {
		return fmt.Errorf("cant create container. Error %v", err)
	}
	deps.InitContainer(container)

	if err = container.Infrastructure.TransactionManager.Launch(); err != nil {
		panic(fmt.Errorf("error launch db:%v", err))
	}

	if err = container.Infrastructure.TransactionManager.Migration(); err != nil {
		panic(fmt.Errorf("error run db migrations:%v", err))
	}

	if err = container.Infrastructure.RedisClient.Launch(ctx); err != nil {
		_ = container.Infrastructure.TransactionManager.Stop()
		panic(fmt.Errorf("error launch redis:%v", err))
	}

	return nil
}

func PostRun(ctx context.Context) {
	//container := deps.Container
	//var err error
}
func OnStop(ctx context.Context) {
	container := deps.Container
	_ = container.Infrastructure.RedisClient.Stop()
	_ = container.Infrastructure.TransactionManager.Stop()
}
