package core_deps

import (
	team_members "mkk_basis/rest_api/internal/app/core/repositorys/team-members"
	"mkk_basis/rest_api/internal/app/core/repositorys/teams"
	"mkk_basis/rest_api/internal/app/core/repositorys/users"
)

type RepositoryDependencies struct {
	UsersRepository       users.UserRepository
	TeamsRepository       teams.TeamRepository
	TeamMembersRepository team_members.TeamMemberRepository
}

func NewRepositoriesDependencies() *RepositoryDependencies {
	return &RepositoryDependencies{
		UsersRepository:       users.NewUserRepository(),
		TeamsRepository:       teams.NewTeamRepository(),
		TeamMembersRepository: team_members.NewTeamMemberRepository(),
	}
}
