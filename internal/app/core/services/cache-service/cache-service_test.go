package cache_service

import (
	"context"
	"errors"
	"testing"
	"time"

	tasks_entities "mkk_basis/rest_api/internal/app/core/entities/tasks-entities"
	redis_service "mkk_basis/rest_api/internal/app/infrastructure/redis-service"
)

type redisClientStub struct {
	values map[string][]byte
	ttl    time.Duration
}

func (s *redisClientStub) Launch(context.Context) error { return nil }
func (s *redisClientStub) Stop() error                  { return nil }

func (s *redisClientStub) Get(_ context.Context, key string) ([]byte, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, redis_service.ErrKeyNotFound
	}
	return value, nil
}

func (s *redisClientStub) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	s.values[key] = value
	s.ttl = ttl
	return nil
}

func (s *redisClientStub) Incr(_ context.Context, key string) (int64, error) {
	current := int64(0)
	if value, ok := s.values[key]; ok {
		if string(value) != "0" {
			return 0, errors.New("unexpected version value")
		}
	}
	current++
	s.values[key] = []byte("1")
	return current, nil
}

func TestTeamTasksCacheLifecycle(t *testing.T) {
	ctx := context.Background()
	redisClient := &redisClientStub{values: make(map[string][]byte)}
	service := NewCacheService(redisClient)
	filter := &tasks_entities.TaskFilterRequest{TeamID: 42}

	_, _, version, found, err := service.GetTeamTasks(ctx, filter)
	if err != nil || found || version != 0 {
		t.Fatalf("expected cache miss with version 0, got found=%v version=%d err=%v", found, version, err)
	}

	tasks := []*tasks_entities.TaskResponse{{ID: 7, TeamID: 42, Title: "cached task"}}
	if err = service.SetTeamTasks(ctx, filter, version, tasks, 1); err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}
	if redisClient.ttl != TeamTasksTTL {
		t.Fatalf("expected TTL %s, got %s", TeamTasksTTL, redisClient.ttl)
	}

	cachedTasks, contentRange, _, found, err := service.GetTeamTasks(ctx, filter)
	if err != nil || !found {
		t.Fatalf("expected cache hit, got found=%v err=%v", found, err)
	}
	if contentRange != 1 || len(cachedTasks) != 1 || cachedTasks[0].ID != 7 {
		t.Fatalf("unexpected cached result: tasks=%+v contentRange=%d", cachedTasks, contentRange)
	}

	if err = service.InvalidateTeamTasks(ctx, filter.TeamID); err != nil {
		t.Fatalf("failed to invalidate cache: %v", err)
	}
	_, _, version, found, err = service.GetTeamTasks(ctx, filter)
	if err != nil || found || version != 1 {
		t.Fatalf("expected cache miss with version 1, got found=%v version=%d err=%v", found, version, err)
	}
}
