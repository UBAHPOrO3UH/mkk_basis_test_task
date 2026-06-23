package tasks_handler

import (
	"context"

	tasks_entities "mkk_basis/rest_api/internal/app/core/entities/tasks-entities"
	tasks_service "mkk_basis/rest_api/internal/app/core/services/tasks-service"
	"mkk_basis/rest_api/internal/app/deps"
)

func CreateTask(
	ctx context.Context,
	userID uint64,
	request *tasks_entities.CreateTaskRequest,
) (*tasks_entities.TaskResponse, error) {
	return deps.Container.Core.Services.TasksService.CreateTask(ctx, userID, request)
}

func GetTasks(
	ctx context.Context,
	userID uint64,
	params *tasks_entities.TaskFilterRequest,
) (*tasks_service.FoundTasksResponse, error) {
	return deps.Container.Core.Services.TasksService.GetTasks(ctx, userID, params)
}

func UpdateTask(
	ctx context.Context,
	taskID, userID uint64,
	request *tasks_entities.UpdateTaskRequest,
) (*tasks_entities.TaskResponse, error) {
	return deps.Container.Core.Services.TasksService.UpdateTask(ctx, taskID, userID, request)
}

func GetTaskHistory(
	ctx context.Context,
	taskID, userID uint64,
) ([]*tasks_entities.TaskHistoryResponse, error) {
	return deps.Container.Core.Services.TasksService.GetTaskHistory(ctx, taskID, userID)
}
