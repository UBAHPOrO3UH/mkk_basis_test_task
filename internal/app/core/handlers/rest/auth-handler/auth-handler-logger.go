package auth_handler

import "mkk_basis/rest_api/internal/logger"

var authLogger = logger.NamedSugar("authLogger", map[string]string{
	"layer": "api",
})
