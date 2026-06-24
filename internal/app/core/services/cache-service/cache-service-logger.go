package cache_service

import "mkk_basis/rest_api/internal/logger"

var cacheLogger = logger.NamedSugar("cacheLogger", map[string]string{
	"layer": "service",
})
