package auth_service

import "mkk_basis/rest_api/internal/logger"

var authLogger = logger.NamedSugar("authLogger", map[string]string{
	"layer": "service",
})
