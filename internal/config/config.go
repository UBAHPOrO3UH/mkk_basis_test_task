package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/spf13/viper"
)

var (
	configPath = "./configs"
	configName = "config"
)

type EmptyError struct {
	nameFields []string
}

func (e *EmptyError) Error() string {
	if len(e.nameFields) <= 1 {
		return fmt.Sprintf("fields %s is empty", strings.Join(e.nameFields, ", "))
	}
	return fmt.Sprintf("fields %s are empty", strings.Join(e.nameFields, ", "))
}

type AppConfig struct {
	Server   *ServerConfig
	AppInfo  *AppInfoConfig
	Database *DataBaseConfig
	Redis    *RedisConfig
	Auth     *AuthConfig
}

type AppInfoConfig struct {
	MaxProcess  int  `env:"APP_MAX_PROCESS"`
	UseProfiler bool `env:"APP_USE_PROFILER"`
	TestMode    bool `env:"APP_TEST_MODE"`
}

type ServerConfig struct {
	Port         int    `env:"SERVER_PORT"               json:"port"               binding:"required"`
	Host         string `env:"SERVER_HOST"               json:"host"               binding:"required"`
	HttpProtocol string `env:"SERVER_HTTP_PROTOCOL"      json:"http_protocol"      binding:"required"`
}

type DataBaseConfig struct {
	DbHost                 string `env:"DB_HOST"                      json:"db_host"     binding:"required"`
	DbUser                 string `env:"DB_USER"                      json:"db_user"     binding:"required"`
	DbPassword             string `env:"DB_PASSWORD"                  json:"db_password" binding:"required"`
	DbName                 string `env:"DB_NAME"                      json:"db_name"     binding:"required"`
	DbPort                 string `env:"DB_PORT"                      json:"db_port"     binding:"required"`
	MaxConnections         int    `env:"DB_MAX_CONN"               json:"-"` // exclude from json
	MaxIdleConns           int    `env:"DB_MAX_IDLE_CONNS"     json:"-"`
	ConnMaxLifetimeMinutes int    `env:"DB_CONN_MAX_LIFETIME_MINUTES"  json:"-"`
	ConnMaxIdleTimeMinutes int    `env:"DB_CONN_MAX_IDLE_TIME_MINUTES" json:"-"`
}

type RedisConfig struct {
	Host                string `env:"REDIS_HOST"                  json:"host"`
	Port                string `env:"REDIS_PORT"                  json:"port"`
	Password            string `env:"REDIS_PASSWORD"              json:"-"`
	DB                  int    `env:"REDIS_DB"                    json:"db"`
	DialTimeoutSeconds  int    `env:"REDIS_DIAL_TIMEOUT_SECONDS"  json:"dial_timeout_seconds"`
	ReadTimeoutSeconds  int    `env:"REDIS_READ_TIMEOUT_SECONDS"  json:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `env:"REDIS_WRITE_TIMEOUT_SECONDS" json:"write_timeout_seconds"`
}

type AuthConfig struct {
	JWTSecret             string `env:"JWT_SECRET"               json:"-"`
	JWTIssuer             string `env:"JWT_ISSUER"               json:"jwt_issuer"`
	AccessTokenTTLMinutes int    `env:"ACCESS_TOKEN_TTL_MINUTES" json:"access_token_ttl_minutes"`
	RefreshTokenTTLHours  int    `env:"REFRESH_TOKEN_TTL_HOURS"  json:"refresh_token_ttl_hours"`
	CookieSecure          bool   `env:"AUTH_COOKIE_SECURE"       json:"cookie_secure"`
	CookieDomain          string `env:"AUTH_COOKIE_DOMAIN"       json:"cookie_domain"`
}

var CurrentConfig = &AppConfig{
	Server: &ServerConfig{
		HttpProtocol: "http",
		Host:         "0.0.0.0",
		Port:         8080,
	},
	AppInfo: &AppInfoConfig{
		MaxProcess:  4,
		UseProfiler: false,
		TestMode:    false,
	},
	Database: &DataBaseConfig{
		DbHost:                 "localhost",
		DbPort:                 "3306",
		DbUser:                 "app",
		DbPassword:             "app",
		DbName:                 "mkk_basis_tasks",
		MaxConnections:         16,
		MaxIdleConns:           8,
		ConnMaxLifetimeMinutes: 30,
		ConnMaxIdleTimeMinutes: 5,
	},
	Redis: &RedisConfig{
		Host:                "localhost",
		Port:                "6379",
		DB:                  0,
		DialTimeoutSeconds:  5,
		ReadTimeoutSeconds:  3,
		WriteTimeoutSeconds: 3,
	},
	Auth: &AuthConfig{
		JWTIssuer:             "mkk-basis-rest-api",
		JWTSecret:             "test",
		AccessTokenTTLMinutes: 15,
		RefreshTokenTTLHours:  24 * 7,
		CookieSecure:          false,
	},
}

func (c *AuthConfig) Validate() error {
	if c == nil {
		return errors.New("auth config is required")
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must contain at least 32 characters")
	}
	if strings.TrimSpace(c.JWTIssuer) == "" {
		return errors.New("JWT_ISSUER must not be empty")
	}
	if c.AccessTokenTTLMinutes <= 0 {
		return errors.New("ACCESS_TOKEN_TTL_MINUTES must be greater than zero")
	}
	if c.RefreshTokenTTLHours <= 0 {
		return errors.New("REFRESH_TOKEN_TTL_HOURS must be greater than zero")
	}

	return nil
}

func ReloadConfig() {
	defer func() {
		jsonConfig, _ := json.Marshal(CurrentConfig)
		configLogger.Debugf("result config is %s %s", "config", string(jsonConfig))
	}()

	err := viper.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			configLogger.DPanicf("failed to read config %s", err)
		}
		configLogger.Debugf(
			"not found file with name %s in path %s; using defaults/env config",
			configName,
			configPath,
		)
	} else {
		err = viper.Unmarshal(&CurrentConfig)
		if err != nil {
			configLogger.DPanicf("failed to parse config %s", err)
		}
	}
	MustApplyEnv()
}

func ApplyEnv() error {
	cfg := reflect.ValueOf(CurrentConfig)
	if cfg.Kind() != reflect.Pointer || cfg.IsNil() {
		return nil
	}
	cfg = cfg.Elem()
	if cfg.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < cfg.NumField(); i++ {
		f := cfg.Field(i)

		if f.Kind() != reflect.Pointer || f.IsNil() {
			continue
		}
		if f.Elem().Kind() != reflect.Struct {
			continue
		}

		if err := cleanenv.ReadEnv(f.Interface()); err != nil {
			return err
		}
	}

	return nil
}

func MustApplyEnv() {
	if err := ApplyEnv(); err != nil {
		panic(err)
	}
}

// ---
func init() {
	viper.AddConfigPath(configPath)
	viper.SetConfigName(configName)
	ReloadConfig()
}
