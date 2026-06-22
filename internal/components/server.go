package components

import "context"

type Server interface {
	Serve(ctx context.Context) error
	Stop(ctx context.Context) error
	GetName() string
}
