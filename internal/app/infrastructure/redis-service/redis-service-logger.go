package redis_service

import "mkk_basis/rest_api/internal/logger"

var redisLogger = logger.NamedSugar("redisLogger", map[string]string{
	"layer": "infrastructure",
})
