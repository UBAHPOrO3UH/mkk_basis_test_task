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
	Server    *ServerConfig    `mapstructure:"server" yaml:"server"`
	AppInfo   *AppInfoConfig   `mapstructure:"app_info" yaml:"app_info"`
	Database  *DataBaseConfig  `mapstructure:"database" yaml:"database"`
	Redis     *RedisConfig     `mapstructure:"redis" yaml:"redis"`
	Auth      *AuthConfig      `mapstructure:"auth" yaml:"auth"`
	RateLimit *RateLimitConfig `mapstructure:"rate_limit" yaml:"rate_limit"`
	Metrics   *MetricsConfig   `mapstructure:"metrics" yaml:"metrics"`
	Email     *EmailConfig     `mapstructure:"email" yaml:"email"`
}

type AppInfoConfig struct {
	MaxProcess  int  `env:"APP_MAX_PROCESS"  mapstructure:"max_process"  yaml:"max_process"`
	UseProfiler bool `env:"APP_USE_PROFILER" mapstructure:"use_profiler" yaml:"use_profiler"`
	TestMode    bool `env:"APP_TEST_MODE"     mapstructure:"test_mode"    yaml:"test_mode"`
}

type ServerConfig struct {
	Port                   int    `env:"SERVER_PORT"                     json:"port"                     mapstructure:"port"                     yaml:"port"                     binding:"required"`
	Host                   string `env:"SERVER_HOST"                     json:"host"                     mapstructure:"host"                     yaml:"host"                     binding:"required"`
	HttpProtocol           string `env:"SERVER_HTTP_PROTOCOL"            json:"http_protocol"            mapstructure:"http_protocol"            yaml:"http_protocol"            binding:"required"`
	ShutdownTimeoutSeconds int    `env:"SERVER_SHUTDOWN_TIMEOUT_SECONDS" json:"shutdown_timeout_seconds" mapstructure:"shutdown_timeout_seconds" yaml:"shutdown_timeout_seconds"`
}

type DataBaseConfig struct {
	DbHost                 string `env:"DB_HOST"                       json:"db_host"     mapstructure:"db_host"                    yaml:"db_host"     binding:"required"`
	DbUser                 string `env:"DB_USER"                       json:"db_user"     mapstructure:"db_user"                    yaml:"db_user"     binding:"required"`
	DbPassword             string `env:"DB_PASSWORD"                   json:"db_password" mapstructure:"db_password"                yaml:"db_password" binding:"required"`
	DbName                 string `env:"DB_NAME"                       json:"db_name"     mapstructure:"db_name"                    yaml:"db_name"     binding:"required"`
	DbPort                 string `env:"DB_PORT"                       json:"db_port"     mapstructure:"db_port"                    yaml:"db_port"     binding:"required"`
	MaxConnections         int    `env:"DB_MAX_CONN"                   json:"-"           mapstructure:"max_connections"            yaml:"max_connections"`
	MaxIdleConns           int    `env:"DB_MAX_IDLE_CONNS"             json:"-"           mapstructure:"max_idle_conns"             yaml:"max_idle_conns"`
	ConnMaxLifetimeMinutes int    `env:"DB_CONN_MAX_LIFETIME_MINUTES"  json:"-"           mapstructure:"conn_max_lifetime_minutes"  yaml:"conn_max_lifetime_minutes"`
	ConnMaxIdleTimeMinutes int    `env:"DB_CONN_MAX_IDLE_TIME_MINUTES" json:"-"           mapstructure:"conn_max_idle_time_minutes" yaml:"conn_max_idle_time_minutes"`
}

type RedisConfig struct {
	Host                string `env:"REDIS_HOST"                  json:"host"                  mapstructure:"host"                  yaml:"host"`
	Port                string `env:"REDIS_PORT"                  json:"port"                  mapstructure:"port"                  yaml:"port"`
	Password            string `env:"REDIS_PASSWORD"              json:"-"                     mapstructure:"password"              yaml:"password"`
	DB                  int    `env:"REDIS_DB"                    json:"db"                    mapstructure:"db"                    yaml:"db"`
	DialTimeoutSeconds  int    `env:"REDIS_DIAL_TIMEOUT_SECONDS"  json:"dial_timeout_seconds"  mapstructure:"dial_timeout_seconds"  yaml:"dial_timeout_seconds"`
	ReadTimeoutSeconds  int    `env:"REDIS_READ_TIMEOUT_SECONDS"  json:"read_timeout_seconds"  mapstructure:"read_timeout_seconds"  yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `env:"REDIS_WRITE_TIMEOUT_SECONDS" json:"write_timeout_seconds" mapstructure:"write_timeout_seconds" yaml:"write_timeout_seconds"`
}

type AuthConfig struct {
	JWTSecret             string `env:"JWT_SECRET"               json:"-"                        mapstructure:"jwt_secret"               yaml:"jwt_secret"`
	JWTIssuer             string `env:"JWT_ISSUER"               json:"jwt_issuer"               mapstructure:"jwt_issuer"               yaml:"jwt_issuer"`
	AccessTokenTTLMinutes int    `env:"ACCESS_TOKEN_TTL_MINUTES" json:"access_token_ttl_minutes" mapstructure:"access_token_ttl_minutes" yaml:"access_token_ttl_minutes"`
	RefreshTokenTTLHours  int    `env:"REFRESH_TOKEN_TTL_HOURS"  json:"refresh_token_ttl_hours"  mapstructure:"refresh_token_ttl_hours"  yaml:"refresh_token_ttl_hours"`
	CookieSecure          bool   `env:"AUTH_COOKIE_SECURE"       json:"cookie_secure"            mapstructure:"cookie_secure"           yaml:"cookie_secure"`
	CookieDomain          string `env:"AUTH_COOKIE_DOMAIN"       json:"cookie_domain"            mapstructure:"cookie_domain"           yaml:"cookie_domain"`
}

type RateLimitConfig struct {
	Enabled           bool `env:"RATE_LIMIT_ENABLED"             mapstructure:"enabled"             yaml:"enabled"`
	RequestsPerMinute int  `env:"RATE_LIMIT_REQUESTS_PER_MINUTE" mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
}

type MetricsConfig struct {
	Enabled bool   `env:"METRICS_ENABLED" mapstructure:"enabled" yaml:"enabled"`
	Path    string `env:"METRICS_PATH"    mapstructure:"path"    yaml:"path"`
}

type EmailConfig struct {
	Enabled        bool                  `env:"EMAIL_ENABLED"         mapstructure:"enabled"         yaml:"enabled"`
	MockFailure    bool                  `env:"EMAIL_MOCK_FAILURE"    mapstructure:"mock_failure"    yaml:"mock_failure"`
	MockLatencyMS  int                   `env:"EMAIL_MOCK_LATENCY_MS" mapstructure:"mock_latency_ms" yaml:"mock_latency_ms"`
	CircuitBreaker *CircuitBreakerConfig `mapstructure:"circuit_breaker" yaml:"circuit_breaker"`
}

type CircuitBreakerConfig struct {
	FailureThreshold   int `env:"EMAIL_CB_FAILURE_THRESHOLD"    mapstructure:"failure_threshold"    yaml:"failure_threshold"`
	OpenTimeoutSeconds int `env:"EMAIL_CB_OPEN_TIMEOUT_SECONDS" mapstructure:"open_timeout_seconds" yaml:"open_timeout_seconds"`
}

var CurrentConfig = &AppConfig{
	Server: &ServerConfig{
		HttpProtocol:           "http",
		Host:                   "0.0.0.0",
		Port:                   8080,
		ShutdownTimeoutSeconds: 30,
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
	RateLimit: &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 100,
	},
	Metrics: &MetricsConfig{
		Enabled: true,
		Path:    "/metrics",
	},
	Email: &EmailConfig{
		Enabled:       true,
		MockFailure:   false,
		MockLatencyMS: 0,
		CircuitBreaker: &CircuitBreakerConfig{
			FailureThreshold:   3,
			OpenTimeoutSeconds: 30,
		},
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
	viper.SetConfigType("yaml")
	ReloadConfig()
}
