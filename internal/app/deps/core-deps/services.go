package core_deps

import (
	auth_service "mkk_basis/rest_api/internal/app/core/services/auth-service"
	cache_service "mkk_basis/rest_api/internal/app/core/services/cache-service"
	tasks_service "mkk_basis/rest_api/internal/app/core/services/tasks-service"
	teams_service "mkk_basis/rest_api/internal/app/core/services/teams-service"
	users_service "mkk_basis/rest_api/internal/app/core/services/users-service"
	infrastructure_deps "mkk_basis/rest_api/internal/app/deps/infrastructure-deps"
	"mkk_basis/rest_api/internal/config"
)

type ServicesDependencies struct {
	UsersService users_service.UserService
	AuthService  auth_service.AuthService
	TeamsService teams_service.TeamService
	TasksService tasks_service.TaskService
	CacheService cache_service.CacheService
}

func NewServicesDependencies(infrastructureDeps *infrastructure_deps.InfrastructureDependencies, repoDeps *RepositoryDependencies) *ServicesDependencies {
	usersService := users_service.NewUserService(infrastructureDeps.TransactionManager, repoDeps.UsersRepository)
	tokenService := auth_service.NewTokenService(config.CurrentConfig.Auth)
	authService := auth_service.NewAuthService(
		infrastructureDeps.TransactionManager, repoDeps.UsersRepository, usersService, tokenService,
	)
	teamsService := teams_service.NewTeamService(
		infrastructureDeps.TransactionManager,
		repoDeps.TeamsRepository,
		repoDeps.TeamMembersRepository,
		repoDeps.UsersRepository,
	)
	cacheService := cache_service.NewCacheService(infrastructureDeps.RedisClient)
	tasksService := tasks_service.NewTaskService(
		infrastructureDeps.TransactionManager,
		repoDeps.TasksRepository,
		repoDeps.TaskHistoryRepository,
		repoDeps.TaskCommentsRepository,
		repoDeps.TeamMembersRepository,
		cacheService,
	)
	return &ServicesDependencies{
		UsersService: usersService,
		AuthService:  authService,
		TeamsService: teamsService,
		TasksService: tasksService,
		CacheService: cacheService,
	}
}
