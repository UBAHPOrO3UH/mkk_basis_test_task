package users_router

import (
	"errors"
	users_entities "mkk_basis/rest_api/internal/app/core/entities/users-entities"
	users_filter "mkk_basis/rest_api/internal/app/core/entities/users-filter"
	users_handler "mkk_basis/rest_api/internal/app/core/handlers/rest/users-handler"
	"mkk_basis/rest_api/internal/common"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func AddRoutes(r *gin.RouterGroup) {
	router := r.Group("/users")
	{
		router.POST("", createUserRoute)
		router.GET("", getUsersRoute)
		router.GET("/filter", getUsersFilterRoute)
		router.GET("/by-email", getUserByEmailRoute)

		router.GET("/:id", getUserRoute)
		router.PUT("/:id", updateUserRoute)
		router.DELETE("/:id", deleteUserRoute)
	}
}

// Create user
//
//	@Summary		Create user
//	@Description	Создание пользователя
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		users_entities.UserRequest	true	"User"
//	@Success		201		{object}	common.APIResponse{result=users_entities.UserResponse}
//	@Router			/api/v1/users [post]
func createUserRoute(c *gin.Context) {
	var request *users_entities.UserRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	user, err := users_handler.CreateUser(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	c.JSON(http.StatusCreated, common.ResultResponseNoErr(user))
}

// Update user
//
//	@Summary		Update user
//	@Description	Обновление пользователя
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int							true	"User ID"
//	@Param			user	body		users_entities.UserRequest	true	"User"
//	@Success		200		{object}	common.APIResponse{result=users_entities.UserResponse}
//	@Router			/api/v1/users/{id} [put]
func updateUserRoute(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	var request *users_entities.UserRequest

	if err = c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	user, err := users_handler.UpdateUser(c.Request.Context(), id, request)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	c.JSON(http.StatusOK, common.ResultResponseNoErr(user))
}

// Get users
//
//	@Summary		Get users
//	@Description	Получить список пользователей без фильтра
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	common.APIResponse{result=[]users_entities.UserResponse}
//	@Router			/api/v1/users [get]
func getUsersRoute(c *gin.Context) {
	users, err := users_handler.GetUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	c.JSON(http.StatusOK, common.ResultResponseNoErr(users))
}

// Get users with filter
//
//	@Summary		Get users with filter
//	@Description	Получить список пользователей с фильтрацией и пагинацией
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			query	query		users_filter.UserFilterRequest	true	"Filter parameters"
//	@Success		200		{object}	common.APIResponse{result=[]users_entities.UserResponse}
//	@Router			/api/v1/users/filter [get]
func getUsersFilterRoute(c *gin.Context) {
	var params users_filter.UserFilterRequest

	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	usersResp, err := users_handler.GetUsersFilter(c.Request.Context(), &params)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	c.Header("Content-Range", strconv.FormatInt(usersResp.ContentRange, 10))
	c.JSON(http.StatusOK, common.ResultResponseNoErr(usersResp.Users))
}

// Get user by id
//
//	@Summary		Get user by id
//	@Description	Получить пользователя по ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	common.APIResponse{result=users_entities.UserResponse}
//	@Router			/api/v1/users/{id} [get]
func getUserRoute(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	user, err := users_handler.GetUserById(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.ResultResponseNoErr(user))
}

// Get user by email
//
//	@Summary		Get user by email
//	@Description	Получить пользователя по email
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			email	query		string	true	"User email"
//	@Success		200		{object}	common.APIResponse{result=users_entities.UserResponse}
//	@Router			/api/v1/users/by-email [get]
func getUserByEmailRoute(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		err := errors.New("email is required")
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	user, err := users_handler.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.ResultResponseNoErr(user))
}

// Delete user by id
//
//	@Summary		Delete user by id
//	@Description	Удалить пользователя по ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	common.APIResponse{result=uint64}
//	@Router			/api/v1/users/{id} [delete]
func deleteUserRoute(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	returnedUserID, err := users_handler.DeleteUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	c.JSON(http.StatusOK, common.ResultResponseNoErr(returnedUserID))
}
