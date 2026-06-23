package database_service

import "mkk_basis/rest_api/internal/logger"

var dbLogger = logger.NamedSugar("dbLogger", map[string]string{
	"layer": "infrastructure",
})
