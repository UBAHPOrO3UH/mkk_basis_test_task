package components

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func AwaitSignal(parent context.Context) context.Context {
	ctx, cancel := context.WithCancel(parent)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-parent.Done():
		case <-sig:
			cancel()
		}
	}()
	return ctx
}
