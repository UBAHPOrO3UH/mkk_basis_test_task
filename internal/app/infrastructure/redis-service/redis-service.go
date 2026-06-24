package redis_service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mkk_basis/rest_api/internal/config"

	"github.com/redis/go-redis/v9"
)

var (
	ErrClientNotStarted = errors.New("redis client is not started")
	ErrKeyNotFound      = errors.New("redis key not found")
)

type RedisClient interface {
	Launch(ctx context.Context) error
	Stop() error
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
}

type RedisClientImpl struct {
	config *config.RedisConfig
	client *redis.Client
}

func NewRedisClient(redisConfig *config.RedisConfig) RedisClient {
	return &RedisClientImpl{config: redisConfig}
}

func (c *RedisClientImpl) Launch(ctx context.Context) error {
	if c.config == nil {
		return errors.New("redis config is required")
	}

	c.client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", c.config.Host, c.config.Port),
		Password:     c.config.Password,
		DB:           c.config.DB,
		DialTimeout:  time.Duration(c.config.DialTimeoutSeconds) * time.Second,
		ReadTimeout:  time.Duration(c.config.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(c.config.WriteTimeoutSeconds) * time.Second,
	})

	if err := c.client.Ping(ctx).Err(); err != nil {
		_ = c.client.Close()
		c.client = nil
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	redisLogger.Infof("redis connected successfully; host=%s port=%s db=%d", c.config.Host, c.config.Port, c.config.DB)
	return nil
}

func (c *RedisClientImpl) Stop() error {
	if c.client == nil {
		return nil
	}

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}
	c.client = nil
	redisLogger.Info("redis connection closed")
	return nil
}

func (c *RedisClientImpl) Get(ctx context.Context, key string) ([]byte, error) {
	if c.client == nil {
		return nil, ErrClientNotStarted
	}

	value, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get redis key %q: %w", key, err)
	}
	return value, nil
}

func (c *RedisClientImpl) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c.client == nil {
		return ErrClientNotStarted
	}

	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set redis key %q: %w", key, err)
	}
	return nil
}

func (c *RedisClientImpl) Incr(ctx context.Context, key string) (int64, error) {
	if c.client == nil {
		return 0, ErrClientNotStarted
	}

	value, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment redis key %q: %w", key, err)
	}
	return value, nil
}
