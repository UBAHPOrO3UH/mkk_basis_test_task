package teams_service

import "mkk_basis/rest_api/internal/logger"

var teamsLogger = logger.NamedSugar("teamsLogger", map[string]string{
	"layer": "service",
})
