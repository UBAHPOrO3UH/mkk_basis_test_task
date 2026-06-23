package teams_handler

import (
	"context"
	teams_entities "mkk_basis/rest_api/internal/app/core/entities/teams-entities"
	"mkk_basis/rest_api/internal/app/deps"
)

func CreateTeam(
	ctx context.Context,
	ownerID uint64,
	request *teams_entities.CreateTeamRequest,
) (*teams_entities.TeamResponse, error) {
	return deps.Container.Core.Services.TeamsService.CreateTeam(ctx, ownerID, request)
}

func GetUserTeams(ctx context.Context, userID uint64) ([]*teams_entities.TeamResponse, error) {
	return deps.Container.Core.Services.TeamsService.GetUserTeams(ctx, userID)
}

func InviteUser(
	ctx context.Context,
	teamID, inviterID uint64,
	request *teams_entities.InviteUserRequest,
) (*teams_entities.TeamMemberResponse, error) {
	return deps.Container.Core.Services.TeamsService.InviteUser(ctx, teamID, inviterID, request)
}
