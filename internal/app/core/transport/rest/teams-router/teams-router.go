package teams_router

import (
	"errors"
	teams_entities "mkk_basis/rest_api/internal/app/core/entities/teams-entities"
	teams_handler "mkk_basis/rest_api/internal/app/core/handlers/rest/teams-handler"
	teams_service "mkk_basis/rest_api/internal/app/core/services/teams-service"
	rest_common "mkk_basis/rest_api/internal/app/core/transport/rest/common"
	"mkk_basis/rest_api/internal/common"
	auth_middleware "mkk_basis/rest_api/internal/components/http/middlewares"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

var teamErrorStatuses = []rest_common.ErrorStatus{
	{Error: teams_service.ErrTeamNameRequired, Status: http.StatusBadRequest},
	{Error: teams_service.ErrInvalidTeamRole, Status: http.StatusBadRequest},
	{Error: teams_service.ErrInsufficientPermission, Status: http.StatusForbidden},
	{Error: teams_service.ErrTeamNotFound, Status: http.StatusNotFound},
	{Error: teams_service.ErrUserNotFound, Status: http.StatusNotFound},
	{Error: teams_service.ErrUserAlreadyTeamMember, Status: http.StatusConflict},
	{Error: teams_service.ErrInvitationEmailFailed, Status: http.StatusServiceUnavailable},
}

func AddRoutes(r *gin.RouterGroup) {
	router := r.Group("/teams")
	router.Use(auth_middleware.AuthMiddleware(), auth_middleware.RateLimitMiddleware())
	{
		router.POST("", createTeamRoute)
		router.GET("", getTeamsRoute)
		router.GET("/stats", getTeamStatsRoute)
		router.POST("/:id/invite", inviteUserRoute)
	}
}

// Create team
//
//	@Summary		Create team
//	@Description	Создать команду; текущий пользователь становится owner
//	@Tags			teams
//	@Accept			json
//	@Produce		json
//	@Param			request	body		teams_entities.CreateTeamRequest	true	"Team data"
//	@Success		201		{object}	common.APIResponse{result=teams_entities.TeamResponse}
//	@Router			/api/v1/teams [post]
func createTeamRoute(c *gin.Context) {
	userID, ok := rest_common.CurrentUserID(c)
	if !ok {
		return
	}

	var request teams_entities.CreateTeamRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	team, err := teams_handler.CreateTeam(c.Request.Context(), userID, &request)
	if err != nil {
		rest_common.WriteError(c, err, teamErrorStatuses...)
		return
	}

	c.JSON(http.StatusCreated, common.ResultResponseNoErr(team))
}

// Get teams
//
//	@Summary		Get user's teams
//	@Description	Получить команды, в которых состоит текущий пользователь
//	@Tags			teams
//	@Produce		json
//	@Success		200	{object}	common.APIResponse{result=[]teams_entities.TeamResponse}
//	@Router			/api/v1/teams [get]
func getTeamsRoute(c *gin.Context) {
	userID, ok := rest_common.CurrentUserID(c)
	if !ok {
		return
	}

	teams, err := teams_handler.GetUserTeams(c.Request.Context(), userID)
	if err != nil {
		rest_common.WriteError(c, err, teamErrorStatuses...)
		return
	}

	c.JSON(http.StatusOK, common.ResultResponseNoErr(teams))
}

// Get team stats
//
//	@Summary		Get team stats
//	@Description	Получить для каждой команды количество участников и выполненных за последние 7 дней задач
//	@Tags			teams
//	@Produce		json
//	@Success		200	{object}	common.APIResponse{result=[]teams_entities.TeamStatsResponse}
//	@Router			/api/v1/teams/stats [get]
func getTeamStatsRoute(c *gin.Context) {
	stats, err := teams_handler.GetTeamStats(c.Request.Context())
	if err != nil {
		rest_common.WriteError(c, err, teamErrorStatuses...)
		return
	}

	c.JSON(http.StatusOK, common.ResultResponseNoErr(stats))
}

// Invite user
//
//	@Summary		Invite user to team
//	@Description	Добавить пользователя в команду; доступно owner и admin
//	@Tags			teams
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int									true	"Team ID"
//	@Param			request	body		teams_entities.InviteUserRequest	true	"Invitation data"
//	@Success		201		{object}	common.APIResponse{result=teams_entities.TeamMemberResponse}
//	@Router			/api/v1/teams/{id}/invite [post]
func inviteUserRoute(c *gin.Context) {
	inviterID, ok := rest_common.CurrentUserID(c)
	if !ok {
		return
	}

	teamID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || teamID == 0 {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(errors.New("invalid team id")))
		return
	}

	var request teams_entities.InviteUserRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	member, err := teams_handler.InviteUser(c.Request.Context(), teamID, inviterID, &request)
	if err != nil {
		rest_common.WriteError(c, err, teamErrorStatuses...)
		return
	}

	c.JSON(http.StatusCreated, common.ResultResponseNoErr(member))
}
