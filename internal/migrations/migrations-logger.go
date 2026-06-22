package migrations

import (
	"mkk_basis/rest_api/internal/logger"

	"go.uber.org/zap"
)

var migrationsLogger *zap.Logger

func init() {
	migrationsLogger = createLogger()
}

func createLogger() *zap.Logger {
	return logger.Logger.Named("migrationsLogger").With(
		zap.String("layer", "server"),
	)
}
