package cache_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	tasks_entities "mkk_basis/rest_api/internal/app/core/entities/tasks-entities"
	redis_service "mkk_basis/rest_api/internal/app/infrastructure/redis-service"
)

const TeamTasksTTL = 5 * time.Minute

type CacheService interface {
	GetTeamTasks(
		ctx context.Context,
		filter *tasks_entities.TaskFilterRequest,
	) (tasks []*tasks_entities.TaskResponse, contentRange, version int64, found bool, err error)
	SetTeamTasks(
		ctx context.Context,
		filter *tasks_entities.TaskFilterRequest,
		version int64,
		tasks []*tasks_entities.TaskResponse,
		contentRange int64,
	) error
	InvalidateTeamTasks(ctx context.Context, teamID uint64) error
}

type CacheServiceImpl struct {
	redisClient redis_service.RedisClient
}

type teamTasksPayload struct {
	Tasks        []*tasks_entities.TaskResponse `json:"tasks"`
	ContentRange int64                          `json:"content_range"`
}

func NewCacheService(redisClient redis_service.RedisClient) CacheService {
	return &CacheServiceImpl{redisClient: redisClient}
}

func (s *CacheServiceImpl) GetTeamTasks(
	ctx context.Context,
	filter *tasks_entities.TaskFilterRequest,
) ([]*tasks_entities.TaskResponse, int64, int64, bool, error) {
	version, err := s.teamTasksVersion(ctx, filter.TeamID)
	if err != nil {
		return nil, 0, 0, false, err
	}

	value, err := s.redisClient.Get(ctx, teamTasksKey(filter, version))
	if errors.Is(err, redis_service.ErrKeyNotFound) {
		cacheLogger.Debugf("team tasks cache miss; team_id=%d version=%d", filter.TeamID, version)
		return nil, 0, version, false, nil
	}
	if err != nil {
		return nil, 0, version, false, err
	}

	var payload teamTasksPayload
	if err = json.Unmarshal(value, &payload); err != nil {
		return nil, 0, version, false, fmt.Errorf("failed to decode team tasks cache: %w", err)
	}

	cacheLogger.Debugf("team tasks cache hit; team_id=%d version=%d", filter.TeamID, version)
	return payload.Tasks, payload.ContentRange, version, true, nil
}

func (s *CacheServiceImpl) SetTeamTasks(
	ctx context.Context,
	filter *tasks_entities.TaskFilterRequest,
	version int64,
	tasks []*tasks_entities.TaskResponse,
	contentRange int64,
) error {
	payload, err := json.Marshal(&teamTasksPayload{
		Tasks:        tasks,
		ContentRange: contentRange,
	})
	if err != nil {
		return fmt.Errorf("failed to encode team tasks cache: %w", err)
	}

	return s.redisClient.Set(ctx, teamTasksKey(filter, version), payload, TeamTasksTTL)
}

func (s *CacheServiceImpl) InvalidateTeamTasks(ctx context.Context, teamID uint64) error {
	version, err := s.redisClient.Incr(ctx, teamTasksVersionKey(teamID))
	if err == nil {
		cacheLogger.Debugf("team tasks cache invalidated; team_id=%d version=%d", teamID, version)
	}
	return err
}

func (s *CacheServiceImpl) teamTasksVersion(ctx context.Context, teamID uint64) (int64, error) {
	value, err := s.redisClient.Get(ctx, teamTasksVersionKey(teamID))
	if errors.Is(err, redis_service.ErrKeyNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	version, err := strconv.ParseInt(string(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid team tasks cache version: %w", err)
	}
	return version, nil
}

func teamTasksVersionKey(teamID uint64) string {
	return fmt.Sprintf("tasks:team:%d:version", teamID)
}

func teamTasksKey(filter *tasks_entities.TaskFilterRequest, version int64) string {
	status := filter.Status
	if status == "" {
		status = "all"
	}

	assigneeID := "all"
	if filter.AssigneeID != nil {
		assigneeID = strconv.FormatUint(*filter.AssigneeID, 10)
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 100
	}

	return fmt.Sprintf(
		"tasks:team:%d:v:%d:status:%s:assignee:%s:limit:%d:shift:%d",
		filter.TeamID,
		version,
		status,
		assigneeID,
		limit,
		filter.Shift,
	)
}
