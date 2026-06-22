package logger

import (
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	envDevelopment = "development"
	envProduction  = "production"
)

var (
	Logger *zap.Logger

	once sync.Once
)

func Init() {
	once.Do(func() {
		l, err := newLogger(Config{
			Environment: getEnv("APP_ENV", envDevelopment),
			Level:       getEnv("LOG_LEVEL", "debug"),
		})
		if err != nil {
			panic(err)
		}

		Logger = l
	})
}

func Sync() {
	if Logger == nil {
		return
	}

	_ = Logger.Sync()
}

type Config struct {
	Environment string
	Level       string
}

func NamedSugar(name string, fields map[string]string) *zap.SugaredLogger {
	Init()

	zapFields := make([]zap.Field, 0, len(fields))

	for key, value := range fields {
		zapFields = append(zapFields, zap.String(key, value))
	}

	return Logger.
		Named(name).
		With(zapFields...).
		Sugar()
}

func Sugar() *zap.SugaredLogger {
	Init()

	return Logger.Sugar()
}

func newLogger(config Config) (*zap.Logger, error) {
	environment := strings.ToLower(strings.TrimSpace(config.Environment))

	var zapConfig zap.Config

	if environment == envProduction {
		zapConfig = zap.NewProductionConfig()
		zapConfig.Encoding = "json"
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.Encoding = "console"
	}

	level, err := parseLevel(config.Level)
	if err != nil {
		return nil, err
	}

	zapConfig.Level.SetLevel(level)

	zapConfig.OutputPaths = []string{"stdout"}
	zapConfig.ErrorOutputPaths = []string{"stderr"}

	zapConfig.EncoderConfig.TimeKey = "time"
	zapConfig.EncoderConfig.LevelKey = "level"
	zapConfig.EncoderConfig.NameKey = "logger"
	zapConfig.EncoderConfig.CallerKey = "caller"
	zapConfig.EncoderConfig.MessageKey = "message"
	zapConfig.EncoderConfig.StacktraceKey = "stacktrace"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	zapConfig.InitialFields = map[string]any{
		"environment": environment,
	}

	return zapConfig.Build()
}

func parseLevel(value string) (zapcore.Level, error) {
	var level zapcore.Level

	if value == "" {
		return zapcore.InfoLevel, nil
	}

	if err := level.Set(strings.ToLower(value)); err != nil {
		return zapcore.InfoLevel, err
	}

	return level, nil
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}
