package tasks_service

import "mkk_basis/rest_api/internal/logger"

var tasksLogger = logger.NamedSugar("tasksLogger", map[string]string{
	"layer": "service",
})
