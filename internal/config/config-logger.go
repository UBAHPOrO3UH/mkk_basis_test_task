package config

import "mkk_basis/rest_api/internal/logger"

var configLogger = logger.NamedSugar("configLogger", map[string]string{"layer": "service"})
