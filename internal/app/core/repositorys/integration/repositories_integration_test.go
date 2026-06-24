//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	tasks_entities "mkk_basis/rest_api/internal/app/core/entities/tasks-entities"
	task_comments "mkk_basis/rest_api/internal/app/core/repositorys/task-comments"
	task_history "mkk_basis/rest_api/internal/app/core/repositorys/task-history"
	"mkk_basis/rest_api/internal/app/core/repositorys/tasks"
	team_members "mkk_basis/rest_api/internal/app/core/repositorys/team-members"
	"mkk_basis/rest_api/internal/app/core/repositorys/teams"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
	database_service "mkk_basis/rest_api/internal/app/infrastructure/database-service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mysqlcontainer "github.com/testcontainers/testcontainers-go/modules/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestRepositoriesWithMySQL(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	container, err := mysqlcontainer.Run(
		ctx,
		"mysql:8.0.46",
		mysqlcontainer.WithDatabase("mkk_basis_tasks"),
		mysqlcontainer.WithUsername("app"),
		mysqlcontainer.WithPassword("app"),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, testcontainers.TerminateContainer(container))
	})

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "loc=UTC", "multiStatements=true")
	require.NoError(t, err)
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	tm := database_service.NewTransactionManager()
	tm.SetConnection(db)
	require.NoError(t, tm.Migration())

	userRepo := users.NewUserRepository()
	teamRepo := teams.NewTeamRepository()
	memberRepo := team_members.NewTeamMemberRepository()
	taskRepo := tasks.NewTaskRepository()
	historyRepo := task_history.NewTaskHistoryRepository()
	commentRepo := task_comments.NewTaskCommentRepository()

	owner, err := userRepo.Create(&users.UserModel{Username: "owner", PasswordHash: "hash", Name: "Owner"}, db)
	require.NoError(t, err)
	member, err := userRepo.Create(&users.UserModel{Username: "member", PasswordHash: "hash", Name: "Member"}, db)
	require.NoError(t, err)
	outsider, err := userRepo.Create(&users.UserModel{Username: "outsider", PasswordHash: "hash", Name: "Outsider"}, db)
	require.NoError(t, err)

	team, err := teamRepo.Create(&teams.TeamModel{Name: "Core", CreatedBy: owner.ID}, db)
	require.NoError(t, err)
	_, err = memberRepo.Create(&team_members.TeamMemberModel{
		TeamID: team.ID,
		UserID: owner.ID,
		Role:   team_members.TeamRoleOwner,
	}, db)
	require.NoError(t, err)
	_, err = memberRepo.Create(&team_members.TeamMemberModel{
		TeamID: team.ID,
		UserID: member.ID,
		Role:   team_members.TeamRoleMember,
	}, db)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Millisecond)
	recent := now.Add(-24 * time.Hour)
	old := now.Add(-14 * 24 * time.Hour)
	task1, err := taskRepo.Create(&tasks.TaskModel{
		TeamID:      team.ID,
		Title:       "recent done",
		Status:      tasks.TaskStatusDone,
		AssigneeID:  &member.ID,
		CreatedBy:   owner.ID,
		CreatedAt:   now.Add(-time.Hour),
		CompletedAt: &recent,
	}, db)
	require.NoError(t, err)
	_, err = taskRepo.Create(&tasks.TaskModel{
		TeamID:    team.ID,
		Title:     "todo",
		Status:    tasks.TaskStatusTodo,
		CreatedBy: member.ID,
		CreatedAt: now.Add(-2 * time.Hour),
	}, db)
	require.NoError(t, err)
	_, err = taskRepo.Create(&tasks.TaskModel{
		TeamID:      team.ID,
		Title:       "old done",
		Status:      tasks.TaskStatusDone,
		CreatedBy:   owner.ID,
		CreatedAt:   now.Add(-3 * time.Hour),
		CompletedAt: &old,
	}, db)
	require.NoError(t, err)
	orphan, err := taskRepo.Create(&tasks.TaskModel{
		TeamID:     team.ID,
		Title:      "outside assignee",
		Status:     tasks.TaskStatusTodo,
		AssigneeID: &outsider.ID,
		CreatedBy:  owner.ID,
		CreatedAt:  now.Add(-4 * time.Hour),
	}, db)
	require.NoError(t, err)

	t.Run("schema indexes", func(t *testing.T) {
		var names []string
		require.NoError(t, db.Raw(`
			SELECT DISTINCT index_name
			FROM information_schema.statistics
			WHERE table_schema = DATABASE() AND table_name = 'tasks'
		`).Scan(&names).Error)

		assert.Contains(t, names, "idx_tasks_created_at_team_created_by")
		assert.Contains(t, names, "idx_tasks_team_id_id")
		assert.Contains(t, names, "idx_tasks_team_status_id")
		assert.Contains(t, names, "idx_tasks_team_assignee_id")
		assert.NotContains(t, names, "idx_tasks_team_status_assignee")
	})

	t.Run("task filtering and pagination", func(t *testing.T) {
		found, err := taskRepo.FindAllWithFilter(&tasks_entities.TaskFilterRequest{
			TeamID: team.ID,
			Status: "done",
			Limit:  1,
		}, db)
		require.NoError(t, err)
		assert.Equal(t, int64(2), found.ContentRange)
		require.Len(t, found.Tasks, 1)
		assert.Equal(t, "old done", found.Tasks[0].Title)
	})

	t.Run("assignee outside team", func(t *testing.T) {
		found, err := taskRepo.FindAllWithAssigneeOutsideTeam(db)
		require.NoError(t, err)
		require.Len(t, found, 1)
		assert.Equal(t, orphan.ID, found[0].ID)
	})

	t.Run("team statistics", func(t *testing.T) {
		stats, err := teamRepo.FindAllWithStats(db)
		require.NoError(t, err)
		require.Len(t, stats, 1)
		assert.Equal(t, int64(2), stats[0].MemberCount)
		assert.Equal(t, int64(1), stats[0].DoneTasksLastSevenDays)
	})

	t.Run("monthly leaders", func(t *testing.T) {
		leaders, err := userRepo.FindTopTaskCreatorsByTeamForMonth(now, db)
		require.NoError(t, err)
		require.Len(t, leaders, 2)
		assert.Equal(t, owner.ID, leaders[0].UserID)
		assert.Equal(t, int64(3), leaders[0].TaskCount)
		assert.Equal(t, member.ID, leaders[1].UserID)
	})

	t.Run("history and comments ordering", func(t *testing.T) {
		first := now.Add(-2 * time.Minute)
		second := now.Add(-time.Minute)
		_, err := historyRepo.CreateBatch([]*task_history.TaskHistoryModel{
			{TaskID: task1.ID, ChangedBy: owner.ID, FieldName: "status", ChangedAt: first},
			{TaskID: task1.ID, ChangedBy: owner.ID, FieldName: "title", ChangedAt: second},
		}, db)
		require.NoError(t, err)
		history, err := historyRepo.FindAllByTaskID(task1.ID, db)
		require.NoError(t, err)
		require.Len(t, history, 2)
		assert.Equal(t, "title", history[0].FieldName)

		_, err = commentRepo.Create(&task_comments.TaskCommentModel{
			TaskID: task1.ID, UserID: owner.ID, Body: "first", CreatedAt: first,
		}, db)
		require.NoError(t, err)
		_, err = commentRepo.Create(&task_comments.TaskCommentModel{
			TaskID: task1.ID, UserID: member.ID, Body: "second", CreatedAt: second,
		}, db)
		require.NoError(t, err)
		comments, err := commentRepo.FindAllByTaskID(task1.ID, db)
		require.NoError(t, err)
		require.Len(t, comments, 2)
		assert.Equal(t, "first", comments[0].Body)
		assert.Equal(t, "second", comments[1].Body)
	})
}
