package http

import (
	"mkk_basis/rest_api/internal/logger"

	"go.uber.org/zap"
)

var serverLogger *zap.Logger

func init() {
	serverLogger = createLogger()
}

func createLogger() *zap.Logger {
	return logger.Logger.Named("httpServerLogger").With(
		zap.String("layer", "http-server"),
	)
}
