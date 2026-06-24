package tasks_service_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	tasks_entities "mkk_basis/rest_api/internal/app/core/entities/tasks-entities"
	task_history "mkk_basis/rest_api/internal/app/core/repositorys/task-history"
	"mkk_basis/rest_api/internal/app/core/repositorys/tasks"
	team_members "mkk_basis/rest_api/internal/app/core/repositorys/team-members"
	tasks_service "mkk_basis/rest_api/internal/app/core/services/tasks-service"
	"mkk_basis/rest_api/internal/mocks"
	"testing"
)

func expectDBRuns(tm *mocks.MockTransactionManager, count int) {
	tm.EXPECT().
		DBRun(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context, *gorm.DB) error, _ ...*sql.TxOptions) error {
			return fn(ctx, &gorm.DB{})
		}).
		Times(count)
}

func str(value string) *string {
	return &value
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func TestCreateTask(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		taskRepo := mocks.NewMockTaskRepository(t)
		historyRepo := mocks.NewMockTaskHistoryRepository(t)
		commentRepo := mocks.NewMockTaskCommentRepository(t)
		memberRepo := mocks.NewMockTeamMemberRepository(t)
		cache := mocks.NewMockCacheService(t)
		expectDBRuns(tm, 1)

		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7, Role: team_members.TeamRoleOwner}, nil).
			Once()
		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(8), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 8, Role: team_members.TeamRoleMember}, nil).
			Once()
		taskRepo.On("Create", mock.MatchedBy(func(model *tasks.TaskModel) bool {
			return model.TeamID == 10 &&
				model.Title == "task" &&
				model.CreatedBy == 7 &&
				model.Status == tasks.TaskStatusTodo &&
				model.AssigneeID != nil &&
				*model.AssigneeID == 8
		}), mock.Anything).
			Return(&tasks.TaskModel{
				ID:         15,
				TeamID:     10,
				Title:      "task",
				Status:     tasks.TaskStatusTodo,
				AssigneeID: uint64Ptr(8),
				CreatedBy:  7,
			}, nil).
			Once()
		cache.On("InvalidateTeamTasks", mock.Anything, uint64(10)).Return(errors.New("cache unavailable")).Once()

		service := tasks_service.NewTaskService(tm, taskRepo, historyRepo, commentRepo, memberRepo, cache)
		result, err := service.CreateTask(context.Background(), 7, &tasks_entities.TaskRequest{
			TeamID:     10,
			Title:      str("  task  "),
			AssigneeID: uint64Ptr(8),
		})

		require.NoError(t, err)
		assert.Equal(t, uint64(15), result.ID)
		assert.Equal(t, "task", result.Title)
		assert.Equal(t, "todo", result.Status)
	})

	t.Run("membership required", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		memberRepo := mocks.NewMockTeamMemberRepository(t)
		expectDBRuns(tm, 1)
		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
			Return((*team_members.TeamMemberModel)(nil), nil).
			Once()

		service := tasks_service.NewTaskService(
			tm,
			mocks.NewMockTaskRepository(t),
			mocks.NewMockTaskHistoryRepository(t),
			mocks.NewMockTaskCommentRepository(t),
			memberRepo,
			nil,
		)
		_, err := service.CreateTask(context.Background(), 7, &tasks_entities.TaskRequest{
			TeamID: 10,
			Title:  str("task"),
		})

		assert.ErrorIs(t, err, tasks_service.ErrTeamMembershipRequired)
	})

	t.Run("validation", func(t *testing.T) {
		service := tasks_service.NewTaskService(nil, nil, nil, nil, nil, nil)

		_, err := service.CreateTask(context.Background(), 1, nil)
		assert.ErrorIs(t, err, tasks_service.ErrTaskTeamRequired)

		_, err = service.CreateTask(context.Background(), 1, &tasks_entities.TaskRequest{TeamID: 10, Title: str(" ")})
		assert.ErrorIs(t, err, tasks_service.ErrTaskTitleRequired)
	})
}

func TestGetTasks(t *testing.T) {
	t.Run("cache hit", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		memberRepo := mocks.NewMockTeamMemberRepository(t)
		cache := mocks.NewMockCacheService(t)
		filter := &tasks_entities.TaskFilterRequest{TeamID: 10}
		expectDBRuns(tm, 1)

		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7}, nil).
			Once()
		cache.On("GetTeamTasks", mock.Anything, filter).
			Return([]*tasks_entities.TaskResponse{{ID: 1, TeamID: 10}}, int64(1), int64(4), true, nil).
			Once()

		service := tasks_service.NewTaskService(
			tm,
			mocks.NewMockTaskRepository(t),
			mocks.NewMockTaskHistoryRepository(t),
			mocks.NewMockTaskCommentRepository(t),
			memberRepo,
			cache,
		)
		result, err := service.GetTasks(context.Background(), 7, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(1), result.ContentRange)
		assert.Equal(t, uint64(1), result.Tasks[0].ID)
	})

	t.Run("cache miss", func(t *testing.T) {
		tm := mocks.NewMockTransactionManager(t)
		taskRepo := mocks.NewMockTaskRepository(t)
		memberRepo := mocks.NewMockTeamMemberRepository(t)
		cache := mocks.NewMockCacheService(t)
		filter := &tasks_entities.TaskFilterRequest{TeamID: 10, Status: "todo"}
		expectDBRuns(tm, 2)

		memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
			Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7}, nil).
			Once()
		cache.On("GetTeamTasks", mock.Anything, filter).
			Return([]*tasks_entities.TaskResponse(nil), int64(0), int64(4), false, nil).
			Once()
		taskRepo.On("FindAllWithFilter", filter, mock.Anything).
			Return(&tasks.FoundTasks{
				Tasks: []*tasks.TaskModel{{
					ID:        2,
					TeamID:    10,
					Title:     "from db",
					Status:    tasks.TaskStatusTodo,
					CreatedBy: 7,
				}},
				ContentRange: 1,
			}, nil).
			Once()
		cache.On("SetTeamTasks", mock.Anything, filter, int64(4), mock.MatchedBy(func(values []*tasks_entities.TaskResponse) bool {
			return len(values) == 1 && values[0].ID == 2
		}), int64(1)).
			Return(errors.New("cache unavailable")).
			Once()

		service := tasks_service.NewTaskService(
			tm,
			taskRepo,
			mocks.NewMockTaskHistoryRepository(t),
			mocks.NewMockTaskCommentRepository(t),
			memberRepo,
			cache,
		)
		result, err := service.GetTasks(context.Background(), 7, filter)

		require.NoError(t, err)
		assert.Equal(t, int64(1), result.ContentRange)
		assert.Equal(t, "from db", result.Tasks[0].Title)
	})

	t.Run("invalid filter", func(t *testing.T) {
		service := tasks_service.NewTaskService(nil, nil, nil, nil, nil, nil)

		_, err := service.GetTasks(context.Background(), 1, nil)
		assert.ErrorIs(t, err, tasks_service.ErrTeamMembershipRequired)

		_, err = service.GetTasks(context.Background(), 1, &tasks_entities.TaskFilterRequest{TeamID: 1, Status: "invalid"})
		assert.ErrorIs(t, err, tasks_service.ErrInvalidTaskStatus)
	})
}

func TestUpdateTask(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	taskRepo := mocks.NewMockTaskRepository(t)
	historyRepo := mocks.NewMockTaskHistoryRepository(t)
	memberRepo := mocks.NewMockTeamMemberRepository(t)
	cache := mocks.NewMockCacheService(t)
	expectDBRuns(tm, 1)

	current := &tasks.TaskModel{
		ID:          5,
		TeamID:      10,
		Title:       "old",
		Status:      tasks.TaskStatusTodo,
		Description: str("old description"),
		AssigneeID:  uint64Ptr(6),
		CreatedBy:   7,
	}
	taskRepo.On("FindByID", uint64(5), mock.Anything).Return(current, nil).Once()
	memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
		Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7, Role: team_members.TeamRoleOwner}, nil).
		Once()
	memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(8), mock.Anything).
		Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 8, Role: team_members.TeamRoleMember}, nil).
		Once()
	taskRepo.On("Update", uint64(5), mock.MatchedBy(func(values map[string]interface{}) bool {
		status, statusOK := values["status"].(tasks.TaskStatus)
		description, descriptionOK := values["description"].(*string)
		assignee, assigneeOK := values["assignee_id"].(*uint64)
		_, completedOK := values["completed_at"]
		return values["title"] == "new" && statusOK && status == tasks.TaskStatusDone &&
			descriptionOK && *description == "new description" &&
			assigneeOK && *assignee == 8 && completedOK
	}), mock.Anything).
		Return(&tasks.TaskModel{
			ID:          5,
			TeamID:      10,
			Title:       "new",
			Description: str("new description"),
			Status:      tasks.TaskStatusDone,
			AssigneeID:  uint64Ptr(8),
			CreatedBy:   7,
		}, nil).
		Once()
	historyRepo.On("CreateBatch", mock.MatchedBy(func(history []*task_history.TaskHistoryModel) bool {
		return len(history) == 4 &&
			history[0].FieldName == "title" &&
			history[1].FieldName == "description" &&
			history[2].FieldName == "assignee_id" &&
			history[3].FieldName == "status"
	}), mock.Anything).
		Return([]*task_history.TaskHistoryModel{}, nil).
		Once()
	cache.On("InvalidateTeamTasks", mock.Anything, uint64(10)).Return(errors.New("cache unavailable")).Once()

	service := tasks_service.NewTaskService(
		tm,
		taskRepo,
		historyRepo,
		mocks.NewMockTaskCommentRepository(t),
		memberRepo,
		cache,
	)
	result, err := service.UpdateTask(context.Background(), 5, 7, &tasks_entities.TaskRequest{
		Title:       str(" new "),
		Description: str("new description"),
		AssigneeID:  uint64Ptr(8),
		Status:      str("done"),
	})

	require.NoError(t, err)
	assert.Equal(t, "new", result.Title)
	assert.Equal(t, "done", result.Status)
}

func TestUpdateTaskWithNoEffectiveChanges(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	taskRepo := mocks.NewMockTaskRepository(t)
	memberRepo := mocks.NewMockTeamMemberRepository(t)
	expectDBRuns(tm, 1)

	description := "same"
	assigneeID := uint64(8)
	current := &tasks.TaskModel{
		ID:          5,
		TeamID:      10,
		Title:       "same",
		Description: &description,
		Status:      tasks.TaskStatusTodo,
		AssigneeID:  &assigneeID,
		CreatedBy:   7,
	}
	taskRepo.On("FindByID", uint64(5), mock.Anything).Return(current, nil).Once()
	memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
		Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7, Role: team_members.TeamRoleMember}, nil).
		Once()
	memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(8), mock.Anything).
		Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 8, Role: team_members.TeamRoleMember}, nil).
		Once()

	service := tasks_service.NewTaskService(
		tm,
		taskRepo,
		mocks.NewMockTaskHistoryRepository(t),
		mocks.NewMockTaskCommentRepository(t),
		memberRepo,
		nil,
	)
	result, err := service.UpdateTask(context.Background(), 5, 7, &tasks_entities.TaskRequest{
		Title:       str("same"),
		Description: &description,
		AssigneeID:  &assigneeID,
		Status:      str("todo"),
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(5), result.ID)
}

func TestUpdateTaskRejectsUnauthorizedMember(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	taskRepo := mocks.NewMockTaskRepository(t)
	memberRepo := mocks.NewMockTeamMemberRepository(t)
	expectDBRuns(tm, 1)

	taskRepo.On("FindByID", uint64(5), mock.Anything).
		Return(&tasks.TaskModel{ID: 5, TeamID: 10, CreatedBy: 7, Status: tasks.TaskStatusTodo}, nil).
		Once()
	memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(9), mock.Anything).
		Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 9, Role: team_members.TeamRoleMember}, nil).
		Once()

	service := tasks_service.NewTaskService(
		tm,
		taskRepo,
		mocks.NewMockTaskHistoryRepository(t),
		mocks.NewMockTaskCommentRepository(t),
		memberRepo,
		nil,
	)
	_, err := service.UpdateTask(context.Background(), 5, 9, &tasks_entities.TaskRequest{Title: str("new")})

	assert.ErrorIs(t, err, tasks_service.ErrInsufficientPermission)
}

func TestGetTaskHistory(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	taskRepo := mocks.NewMockTaskRepository(t)
	historyRepo := mocks.NewMockTaskHistoryRepository(t)
	memberRepo := mocks.NewMockTeamMemberRepository(t)
	expectDBRuns(tm, 1)

	taskRepo.On("FindByID", uint64(5), mock.Anything).
		Return(&tasks.TaskModel{ID: 5, TeamID: 10}, nil).
		Once()
	memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
		Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7}, nil).
		Once()
	historyRepo.On("FindAllByTaskID", uint64(5), mock.Anything).
		Return([]*task_history.TaskHistoryModel{{ID: 3, TaskID: 5, ChangedBy: 7, FieldName: "status"}}, nil).
		Once()

	service := tasks_service.NewTaskService(
		tm,
		taskRepo,
		historyRepo,
		mocks.NewMockTaskCommentRepository(t),
		memberRepo,
		nil,
	)
	result, err := service.GetTaskHistory(context.Background(), 5, 7)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, uint64(3), result[0].ID)
}

func TestGetTasksWithAssigneeOutsideTeam(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	taskRepo := mocks.NewMockTaskRepository(t)
	expectDBRuns(tm, 1)
	taskRepo.On("FindAllWithAssigneeOutsideTeam", mock.Anything).
		Return([]*tasks.TaskModel{{ID: 1, TeamID: 10, Title: "orphan", Status: tasks.TaskStatusTodo}}, nil).
		Once()

	service := tasks_service.NewTaskService(
		tm,
		taskRepo,
		mocks.NewMockTaskHistoryRepository(t),
		mocks.NewMockTaskCommentRepository(t),
		mocks.NewMockTeamMemberRepository(t),
		nil,
	)
	result, err := service.GetTasksWithAssigneeOutsideTeam(context.Background())

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "orphan", result[0].Title)
}

func TestUpdateTaskValidation(t *testing.T) {
	service := tasks_service.NewTaskService(nil, nil, nil, nil, nil, nil)

	_, err := service.UpdateTask(context.Background(), 1, 1, nil)
	assert.ErrorIs(t, err, tasks_service.ErrNoTaskChanges)

	_, err = service.UpdateTask(context.Background(), 1, 1, &tasks_entities.TaskRequest{Status: str("invalid")})
	assert.ErrorIs(t, err, tasks_service.ErrInvalidTaskStatus)

	_, err = service.UpdateTask(context.Background(), 1, 1, &tasks_entities.TaskRequest{})
	assert.True(t, errors.Is(err, tasks_service.ErrNoTaskChanges))
}

func TestCreateTaskRejectsNonMemberAssignee(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	memberRepo := mocks.NewMockTeamMemberRepository(t)
	expectDBRuns(tm, 1)
	memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
		Return(&team_members.TeamMemberModel{TeamID: 10, UserID: 7}, nil).
		Once()
	memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(8), mock.Anything).
		Return((*team_members.TeamMemberModel)(nil), nil).
		Once()

	service := tasks_service.NewTaskService(
		tm,
		mocks.NewMockTaskRepository(t),
		mocks.NewMockTaskHistoryRepository(t),
		mocks.NewMockTaskCommentRepository(t),
		memberRepo,
		nil,
	)
	_, err := service.CreateTask(context.Background(), 7, &tasks_entities.TaskRequest{
		TeamID:     10,
		Title:      str("task"),
		AssigneeID: uint64Ptr(8),
	})

	assert.ErrorIs(t, err, tasks_service.ErrAssigneeNotTeamMember)
}

func TestGetTasksMembershipFailure(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	memberRepo := mocks.NewMockTeamMemberRepository(t)
	expectDBRuns(tm, 1)
	memberRepo.On("FindByTeamIDAndUserID", uint64(10), uint64(7), mock.Anything).
		Return((*team_members.TeamMemberModel)(nil), nil).
		Once()

	service := tasks_service.NewTaskService(
		tm,
		mocks.NewMockTaskRepository(t),
		mocks.NewMockTaskHistoryRepository(t),
		mocks.NewMockTaskCommentRepository(t),
		memberRepo,
		nil,
	)
	_, err := service.GetTasks(context.Background(), 7, &tasks_entities.TaskFilterRequest{TeamID: 10})

	assert.ErrorIs(t, err, tasks_service.ErrTeamMembershipRequired)
}

func TestGetTaskHistoryNotFound(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	taskRepo := mocks.NewMockTaskRepository(t)
	expectDBRuns(tm, 1)
	taskRepo.On("FindByID", uint64(99), mock.Anything).
		Return((*tasks.TaskModel)(nil), nil).
		Once()

	service := tasks_service.NewTaskService(
		tm,
		taskRepo,
		mocks.NewMockTaskHistoryRepository(t),
		mocks.NewMockTaskCommentRepository(t),
		mocks.NewMockTeamMemberRepository(t),
		nil,
	)
	_, err := service.GetTaskHistory(context.Background(), 99, 7)

	assert.ErrorIs(t, err, tasks_service.ErrTaskNotFound)
}

func TestGetTasksWithAssigneeOutsideTeamError(t *testing.T) {
	tm := mocks.NewMockTransactionManager(t)
	taskRepo := mocks.NewMockTaskRepository(t)
	expectDBRuns(tm, 1)
	expectedErr := errors.New("database unavailable")
	taskRepo.On("FindAllWithAssigneeOutsideTeam", mock.Anything).
		Return([]*tasks.TaskModel(nil), expectedErr).
		Once()

	service := tasks_service.NewTaskService(
		tm,
		taskRepo,
		mocks.NewMockTaskHistoryRepository(t),
		mocks.NewMockTaskCommentRepository(t),
		mocks.NewMockTeamMemberRepository(t),
		nil,
	)
	_, err := service.GetTasksWithAssigneeOutsideTeam(context.Background())

	assert.ErrorIs(t, err, expectedErr)
}
