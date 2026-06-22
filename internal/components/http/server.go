package http

import (
	"context"
	"fmt"
	"mkk_basis/rest_api/internal/components"
	"mkk_basis/rest_api/internal/config"
	"net/http"
)

type Server struct {
	Name    string
	httpSrv *http.Server
}

func NewHTTPServer() components.Server {
	serverLogger.Info("Create new HTTPServer")
	srvConf := config.CurrentConfig.Server

	engine := GetRoutes()

	s := &Server{
		Name: "HTTPServer",
		httpSrv: &http.Server{
			Addr:    fmt.Sprintf("%s:%d", srvConf.Host, srvConf.Port),
			Handler: engine,
		},
	}

	return s
}

func (s *Server) Serve(ctx context.Context) error {
	serverLogger.Info("Serving HTTPServer")
	defer serverLogger.Info("Finished serving HTTPServer")
	go func() {
		<-ctx.Done()
		_ = s.Stop(context.Background())
	}()
	if err := s.httpSrv.ListenAndServe(); err != nil {
		return fmt.Errorf("http listen: %w", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	serverLogger.Info("Stop HTTPServer")
	defer serverLogger.Info("Stopped HTTPServer")
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}

	return nil
}

func (s *Server) GetName() string {
	return s.Name
}
