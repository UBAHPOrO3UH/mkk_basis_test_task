package core_deps

import (
	task_comments "mkk_basis/rest_api/internal/app/core/repositorys/task-comments"
	task_history "mkk_basis/rest_api/internal/app/core/repositorys/task-history"
	"mkk_basis/rest_api/internal/app/core/repositorys/tasks"
	team_members "mkk_basis/rest_api/internal/app/core/repositorys/team-members"
	"mkk_basis/rest_api/internal/app/core/repositorys/teams"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
)

type RepositoryDependencies struct {
	UsersRepository        users.UserRepository
	TeamsRepository        teams.TeamRepository
	TeamMembersRepository  team_members.TeamMemberRepository
	TasksRepository        tasks.TaskRepository
	TaskHistoryRepository  task_history.TaskHistoryRepository
	TaskCommentsRepository task_comments.TaskCommentRepository
}

func NewRepositoriesDependencies() *RepositoryDependencies {
	return &RepositoryDependencies{
		UsersRepository:        users.NewUserRepository(),
		TeamsRepository:        teams.NewTeamRepository(),
		TeamMembersRepository:  team_members.NewTeamMemberRepository(),
		TasksRepository:        tasks.NewTaskRepository(),
		TaskHistoryRepository:  task_history.NewTaskHistoryRepository(),
		TaskCommentsRepository: task_comments.NewTaskCommentRepository(),
	}
}
