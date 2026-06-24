package auth_router

import (
	"errors"
	auth_entities "mkk_basis/rest_api/internal/app/core/entities/auth-entities"
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	auth_handler "mkk_basis/rest_api/internal/app/core/handlers/rest/auth-handler"
	auth_service "mkk_basis/rest_api/internal/app/core/services/auth-service"
	users_service "mkk_basis/rest_api/internal/app/core/services/users-service"
	"mkk_basis/rest_api/internal/common"
	auth_middleware "mkk_basis/rest_api/internal/components/http/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AddRoutes(router *gin.RouterGroup) {
	router.POST("/register", registerRoute)
	router.POST("/login", loginRoute)
}

// Register
//
//	@Summary		Register
//	@Description	Регистрация нового пользователя
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		users_entities.UserRequest	true	"Registration data"
//	@Success		201		{object}	common.APIResponse
//	@Failure		400		{object}	common.APIResponse
//	@Failure		409		{object}	common.APIResponse
//	@Router			/api/v1/register [post]
func registerRoute(c *gin.Context) {
	var request *users_entities.UserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	user, err := auth_handler.Register(c.Request.Context(), request)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, users_service.ErrUserPasswordRequired) {
			status = http.StatusBadRequest
		} else if errors.Is(err, users_service.ErrUserAlreadyExists) {
			status = http.StatusConflict
		}
		c.JSON(status, common.ErrorResponse(err))
		return
	}

	c.JSON(http.StatusCreated, common.ResultResponseNoErr(user))
}

// Login
//
//	@Summary		Login
//	@Description	Аутентификация; JWT устанавливаются в HttpOnly cookies
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		auth_entities.LoginRequest	true	"Credentials"
//	@Success		200		{object}	common.APIResponse{result=auth_entities.AuthResponse}
//	@Failure		400		{object}	common.APIResponse
//	@Failure		401		{object}	common.APIResponse
//	@Router			/api/v1/login [post]
func loginRoute(c *gin.Context) {
	var request *auth_entities.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	user, tokens, err := auth_handler.Login(c.Request.Context(), request)
	if err != nil {
		if errors.Is(err, auth_service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, common.ErrorResponse(err))
			return
		}
		c.JSON(http.StatusInternalServerError, common.ErrorResponse(errors.New("authentication failed")))
		return
	}

	auth_middleware.SetAuthCookies(c, tokens)
	c.JSON(http.StatusOK, common.ResultResponseNoErr(&auth_entities.AuthResponse{
		User:             user,
		AccessExpiresAt:  tokens.AccessExpiresAt,
		RefreshExpiresAt: tokens.RefreshExpiresAt,
	}))
}
