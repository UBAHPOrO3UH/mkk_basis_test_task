package users_service

import "mkk_basis/rest_api/internal/logger"

var usersLogger = logger.NamedSugar("usersLogger", map[string]string{
	"layer": "service",
})
