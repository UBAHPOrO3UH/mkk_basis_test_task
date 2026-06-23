package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mkk_basis/rest_api/cmd/rest-api/components"
	"mkk_basis/rest_api/internal/app/lifespan"
	internalComponents "mkk_basis/rest_api/internal/components"
	http_server "mkk_basis/rest_api/internal/components/http"
	"mkk_basis/rest_api/internal/config"
	"net/http"
	"os"
	"runtime"
	"time"
)

func Run(ctx context.Context, errSet *components.ErrSet) error {
	runtime.GOMAXPROCS(config.CurrentConfig.AppInfo.MaxProcess)
	var err error
	err = lifespan.PreRun(ctx, false)
	if err != nil {
		return err
	}
	postRun := func(ctx context.Context) {
		lifespan.PostRun(ctx)
	}

	httpServer := http_server.NewHTTPServer()
	if ctx, err = components.Serve(ctx, errSet, postRun, httpServer); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("cant serve httpServer: %w", err)
	}

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	lifespan.OnStop(shutdownCtx)

	return ctx.Err()
}

// @Version	1.0
// @Title		rest-API
func main() {
	ctx := internalComponents.AwaitSignal(context.Background())

	logger := log.New(os.Stderr, "", 0)
	logger.Print("Started")

	errSet := &components.ErrSet{}

	errSet.Add(Run(ctx, errSet))
	if errSet.Error() != nil {
		logger.Fatal(errSet.Error())
	}
}
