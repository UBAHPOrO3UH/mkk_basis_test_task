package tasks_service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	tasks_entities "mkk_basis/rest_api/internal/app/core/entities/tasks-entities"
	task_comments "mkk_basis/rest_api/internal/app/core/repositorys/task-comments"
	task_history "mkk_basis/rest_api/internal/app/core/repositorys/task-history"
	"mkk_basis/rest_api/internal/app/core/repositorys/tasks"
	team_members "mkk_basis/rest_api/internal/app/core/repositorys/team-members"
	cache_service "mkk_basis/rest_api/internal/app/core/services/cache-service"
	database_service "mkk_basis/rest_api/internal/app/infrastructure/database-service"

	"gorm.io/gorm"
)

var (
	ErrTaskNotFound           = errors.New("task not found")
	ErrTeamMembershipRequired = errors.New("team membership required")
	ErrAssigneeNotTeamMember  = errors.New("assignee is not a team member")
	ErrInsufficientPermission = errors.New("insufficient permission")
	ErrTaskTeamRequired       = errors.New("task team id is required")
	ErrTaskTitleRequired      = errors.New("task title is required")
	ErrInvalidTaskStatus      = errors.New("invalid task status")
	ErrNoTaskChanges          = errors.New("no task changes provided")
)

type FoundTasksResponse struct {
	Tasks        []*tasks_entities.TaskResponse
	ContentRange int64
}

type TaskService interface {
	CreateTask(ctx context.Context, userID uint64, request *tasks_entities.TaskRequest) (*tasks_entities.TaskResponse, error)
	GetTasks(ctx context.Context, userID uint64, params *tasks_entities.TaskFilterRequest) (*FoundTasksResponse, error)
	GetTasksWithAssigneeOutsideTeam(ctx context.Context) ([]*tasks_entities.TaskResponse, error)
	UpdateTask(ctx context.Context, taskID, userID uint64, request *tasks_entities.TaskRequest) (*tasks_entities.TaskResponse, error)
	GetTaskHistory(ctx context.Context, taskID, userID uint64) ([]*tasks_entities.TaskHistoryResponse, error)
}

type TaskServiceImpl struct {
	tm                    database_service.TransactionManager
	taskRepository        tasks.TaskRepository
	taskHistoryRepository task_history.TaskHistoryRepository
	taskCommentRepository task_comments.TaskCommentRepository
	teamMemberRepository  team_members.TeamMemberRepository
	cacheService          cache_service.CacheService
}

func NewTaskService(
	tm database_service.TransactionManager,
	taskRepository tasks.TaskRepository,
	taskHistoryRepository task_history.TaskHistoryRepository,
	taskCommentRepository task_comments.TaskCommentRepository,
	teamMemberRepository team_members.TeamMemberRepository,
	cacheService cache_service.CacheService,
) TaskService {
	return &TaskServiceImpl{
		tm:                    tm,
		taskRepository:        taskRepository,
		taskHistoryRepository: taskHistoryRepository,
		taskCommentRepository: taskCommentRepository,
		teamMemberRepository:  teamMemberRepository,
		cacheService:          cacheService,
	}
}

func (s *TaskServiceImpl) CreateTask(
	ctx context.Context,
	userID uint64,
	request *tasks_entities.TaskRequest,
) (*tasks_entities.TaskResponse, error) {
	if request == nil || request.TeamID == 0 {
		return nil, ErrTaskTeamRequired
	}
	if request.Title == nil || strings.TrimSpace(*request.Title) == "" {
		return nil, ErrTaskTitleRequired
	}

	var createdTask *tasks.TaskModel
	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		if err := s.requireTeamMember(request.TeamID, userID, tx); err != nil {
			return err
		}

		assigneeID, err := s.validAssignee(request.TeamID, request.AssigneeID, tx)
		if err != nil {
			return err
		}

		createdTask, err = s.taskRepository.Create(&tasks.TaskModel{
			TeamID:      request.TeamID,
			Title:       strings.TrimSpace(*request.Title),
			Description: request.Description,
			Status:      tasks.TaskStatusTodo,
			AssigneeID:  assigneeID,
			CreatedBy:   userID,
		}, tx)
		return err
	})
	if err != nil {
		tasksLogger.Errorf("failed to create task team_id=%d user_id=%d: %v", request.TeamID, userID, err)
		return nil, err
	}

	s.invalidateTeamTasksCache(ctx, createdTask.TeamID)
	tasksLogger.Infof("task created id=%d team_id=%d user_id=%d", createdTask.ID, createdTask.TeamID, userID)
	return taskResponse(createdTask), nil
}

func (s *TaskServiceImpl) GetTasks(
	ctx context.Context,
	userID uint64,
	params *tasks_entities.TaskFilterRequest,
) (*FoundTasksResponse, error) {
	if params == nil || params.TeamID == 0 {
		return nil, ErrTeamMembershipRequired
	}
	if !validTaskStatus(params.Status, true) {
		return nil, ErrInvalidTaskStatus
	}

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		return s.requireTeamMember(params.TeamID, userID, tx)
	})
	if err != nil {
		tasksLogger.Errorf("failed to check team membership team_id=%d user_id=%d: %v", params.TeamID, userID, err)
		return nil, err
	}

	var cacheVersion int64
	if s.cacheService != nil {
		cachedTasks, contentRange, version, found, cacheErr := s.cacheService.GetTeamTasks(ctx, params)
		cacheVersion = version
		if cacheErr != nil {
			tasksLogger.Warnf("failed to get team tasks from cache team_id=%d: %v", params.TeamID, cacheErr)
		} else if found {
			return &FoundTasksResponse{Tasks: cachedTasks, ContentRange: contentRange}, nil
		}
	}

	response := &FoundTasksResponse{Tasks: make([]*tasks_entities.TaskResponse, 0)}
	err = s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		found, err := s.taskRepository.FindAllWithFilter(params, tx)
		if err != nil {
			return err
		}

		response.Tasks = make([]*tasks_entities.TaskResponse, 0, len(found.Tasks))
		for _, model := range found.Tasks {
			response.Tasks = append(response.Tasks, taskResponse(model))
		}
		response.ContentRange = found.ContentRange
		return nil
	})
	if err != nil {
		tasksLogger.Errorf("failed to get tasks team_id=%d user_id=%d: %v", params.TeamID, userID, err)
		return nil, err
	}

	if s.cacheService != nil {
		if cacheErr := s.cacheService.SetTeamTasks(
			ctx,
			params,
			cacheVersion,
			response.Tasks,
			response.ContentRange,
		); cacheErr != nil {
			tasksLogger.Warnf("failed to set team tasks cache team_id=%d: %v", params.TeamID, cacheErr)
		}
	}

	return response, nil
}

func (s *TaskServiceImpl) GetTasksWithAssigneeOutsideTeam(
	ctx context.Context,
) ([]*tasks_entities.TaskResponse, error) {
	response := make([]*tasks_entities.TaskResponse, 0)

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		models, err := s.taskRepository.FindAllWithAssigneeOutsideTeam(tx)
		if err != nil {
			return err
		}

		response = make([]*tasks_entities.TaskResponse, 0, len(models))
		for _, model := range models {
			response = append(response, taskResponse(model))
		}
		return nil
	})
	if err != nil {
		tasksLogger.Errorf("failed to get tasks with assignee outside team: %v", err)
		return nil, err
	}

	return response, nil
}

func (s *TaskServiceImpl) UpdateTask(
	ctx context.Context,
	taskID, userID uint64,
	request *tasks_entities.TaskRequest,
) (*tasks_entities.TaskResponse, error) {
	if request == nil || !hasTaskChanges(request) {
		return nil, ErrNoTaskChanges
	}
	if request.Status != nil && !validTaskStatus(*request.Status, false) {
		return nil, ErrInvalidTaskStatus
	}

	var (
		updatedTask *tasks.TaskModel
		taskChanged bool
	)
	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		currentTask, err := s.taskRepository.FindByID(taskID, tx)
		if err != nil {
			return err
		}
		if currentTask == nil {
			return ErrTaskNotFound
		}

		member, err := s.teamMemberRepository.FindByTeamIDAndUserID(currentTask.TeamID, userID, tx)
		if err != nil {
			return err
		}
		if member == nil {
			return ErrTeamMembershipRequired
		}
		if !canUpdateTask(currentTask, member, userID, request) {
			return ErrInsufficientPermission
		}

		values := make(map[string]interface{})
		history := make([]*task_history.TaskHistoryModel, 0, 4)

		if request.Title != nil {
			title := strings.TrimSpace(*request.Title)
			if title == "" {
				return ErrTaskTitleRequired
			}
			if title != currentTask.Title {
				values["title"] = title
				history = append(history, historyEntry(taskID, userID, "title", stringValue(currentTask.Title), stringValue(title)))
			}
		}

		if request.Description != nil && !stringPointersEqual(currentTask.Description, request.Description) {
			values["description"] = request.Description
			history = append(history, historyEntry(taskID, userID, "description", currentTask.Description, request.Description))
		}

		if request.AssigneeID != nil {
			assigneeID, err := s.validAssignee(currentTask.TeamID, request.AssigneeID, tx)
			if err != nil {
				return err
			}
			if !uint64PointersEqual(currentTask.AssigneeID, assigneeID) {
				values["assignee_id"] = assigneeID
				history = append(history, historyEntry(
					taskID,
					userID,
					"assignee_id",
					uint64Value(currentTask.AssigneeID),
					uint64Value(assigneeID),
				))
			}
		}

		if request.Status != nil {
			status := tasks.TaskStatus(*request.Status)
			if status != currentTask.Status {
				values["status"] = status
				history = append(history, historyEntry(
					taskID,
					userID,
					"status",
					stringValue(string(currentTask.Status)),
					stringValue(string(status)),
				))
				if status == tasks.TaskStatusDone {
					completedAt := time.Now().UTC()
					values["completed_at"] = &completedAt
				} else {
					values["completed_at"] = nil
				}
			}
		}

		if len(values) == 0 {
			updatedTask = currentTask
			return nil
		}

		updatedTask, err = s.taskRepository.Update(taskID, values, tx)
		if err != nil {
			return err
		}
		taskChanged = true
		_, err = s.taskHistoryRepository.CreateBatch(history, tx)
		return err
	})
	if err != nil {
		tasksLogger.Errorf("failed to update task id=%d user_id=%d: %v", taskID, userID, err)
		return nil, err
	}

	if taskChanged {
		s.invalidateTeamTasksCache(ctx, updatedTask.TeamID)
	}
	tasksLogger.Infof("task updated id=%d user_id=%d", taskID, userID)
	return taskResponse(updatedTask), nil
}

func (s *TaskServiceImpl) GetTaskHistory(
	ctx context.Context,
	taskID, userID uint64,
) ([]*tasks_entities.TaskHistoryResponse, error) {
	response := make([]*tasks_entities.TaskHistoryResponse, 0)

	err := s.tm.DBRun(ctx, func(ctx context.Context, tx *gorm.DB) error {
		task, err := s.taskRepository.FindByID(taskID, tx)
		if err != nil {
			return err
		}
		if task == nil {
			return ErrTaskNotFound
		}
		if err = s.requireTeamMember(task.TeamID, userID, tx); err != nil {
			return err
		}

		models, err := s.taskHistoryRepository.FindAllByTaskID(taskID, tx)
		if err != nil {
			return err
		}
		response = make([]*tasks_entities.TaskHistoryResponse, 0, len(models))
		for _, model := range models {
			response = append(response, taskHistoryResponse(model))
		}
		return nil
	})
	if err != nil {
		tasksLogger.Errorf("failed to get task history task_id=%d user_id=%d: %v", taskID, userID, err)
		return nil, err
	}

	return response, nil
}

func (s *TaskServiceImpl) requireTeamMember(teamID, userID uint64, tx *gorm.DB) error {
	member, err := s.teamMemberRepository.FindByTeamIDAndUserID(teamID, userID, tx)
	if err != nil {
		return err
	}
	if member == nil {
		return ErrTeamMembershipRequired
	}
	return nil
}

func (s *TaskServiceImpl) validAssignee(teamID uint64, assigneeID *uint64, tx *gorm.DB) (*uint64, error) {
	if assigneeID == nil || *assigneeID == 0 {
		return nil, nil
	}
	member, err := s.teamMemberRepository.FindByTeamIDAndUserID(teamID, *assigneeID, tx)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrAssigneeNotTeamMember
	}
	return assigneeID, nil
}

func (s *TaskServiceImpl) invalidateTeamTasksCache(ctx context.Context, teamID uint64) {
	if s.cacheService == nil {
		return
	}
	if err := s.cacheService.InvalidateTeamTasks(ctx, teamID); err != nil {
		tasksLogger.Warnf("failed to invalidate team tasks cache team_id=%d: %v", teamID, err)
	}
}

func canUpdateTask(
	task *tasks.TaskModel,
	member *team_members.TeamMemberModel,
	userID uint64,
	request *tasks_entities.TaskRequest,
) bool {
	if member.Role == team_members.TeamRoleOwner || member.Role == team_members.TeamRoleAdmin || task.CreatedBy == userID {
		return true
	}

	isAssignee := task.AssigneeID != nil && *task.AssigneeID == userID
	statusOnly := request.Status != nil && request.Title == nil && request.Description == nil && request.AssigneeID == nil
	return isAssignee && statusOnly
}

func hasTaskChanges(request *tasks_entities.TaskRequest) bool {
	return request.Title != nil || request.Description != nil || request.Status != nil || request.AssigneeID != nil
}

func validTaskStatus(status string, allowEmpty bool) bool {
	if allowEmpty && status == "" {
		return true
	}
	switch tasks.TaskStatus(status) {
	case tasks.TaskStatusTodo, tasks.TaskStatusInProgress, tasks.TaskStatusDone:
		return true
	default:
		return false
	}
}

func historyEntry(taskID, userID uint64, fieldName string, oldValue, newValue *string) *task_history.TaskHistoryModel {
	return &task_history.TaskHistoryModel{
		TaskID:    taskID,
		ChangedBy: userID,
		FieldName: fieldName,
		OldValue:  oldValue,
		NewValue:  newValue,
	}
}

func taskResponse(model *tasks.TaskModel) *tasks_entities.TaskResponse {
	return &tasks_entities.TaskResponse{
		ID:          model.ID,
		TeamID:      model.TeamID,
		Title:       model.Title,
		Description: model.Description,
		Status:      string(model.Status),
		AssigneeID:  model.AssigneeID,
		CreatedBy:   model.CreatedBy,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		CompletedAt: model.CompletedAt,
	}
}

func taskHistoryResponse(model *task_history.TaskHistoryModel) *tasks_entities.TaskHistoryResponse {
	return &tasks_entities.TaskHistoryResponse{
		ID:        model.ID,
		TaskID:    model.TaskID,
		ChangedBy: model.ChangedBy,
		FieldName: model.FieldName,
		OldValue:  model.OldValue,
		NewValue:  model.NewValue,
		ChangedAt: model.ChangedAt,
	}
}

func stringValue(value string) *string {
	return &value
}

func uint64Value(value *uint64) *string {
	if value == nil {
		return nil
	}
	result := strconv.FormatUint(*value, 10)
	return &result
}

func stringPointersEqual(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func uint64PointersEqual(left, right *uint64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
