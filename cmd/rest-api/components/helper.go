package components

import (
	"context"
	"errors"
	"fmt"
	"mkk_basis/rest_api/internal/components"
	"net/http"
	"sync"
)

func Serve(
	ctx context.Context,
	errSet *ErrSet,
	onStarted func(ctx context.Context),
	servers ...components.Server,
) (context.Context, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		if onStarted != nil {
			onStarted(ctx)
		}

		wg := sync.WaitGroup{}
		defer wg.Wait()
		for _, server := range servers {
			wg.Add(1)
			go func(srv components.Server) {
				defer wg.Done()
				if err := srv.Serve(ctx); err != nil &&
					!errors.Is(err, http.ErrServerClosed) {
					errSet.Add(fmt.Errorf("err serving %q: %w", server.GetName(), err))
					cancel()
				}
			}(server)
		}
		wg.Wait()
		cancel()
	}()

	return ctx, nil
}
