package middlewares

import "mkk_basis/rest_api/internal/logger"

var middleWareLogger = logger.NamedSugar("middleWareLogger", map[string]string{
	"layer": "middleware",
})
