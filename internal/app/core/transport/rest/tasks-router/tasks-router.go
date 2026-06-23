package tasks_router

import (
	tasks_entities "mkk_basis/rest_api/internal/app/core/entities/tasks-entities"
	tasks_handler "mkk_basis/rest_api/internal/app/core/handlers/rest/tasks-handler"
	tasks_service "mkk_basis/rest_api/internal/app/core/services/tasks-service"
	rest_common "mkk_basis/rest_api/internal/app/core/transport/rest/common"
	"mkk_basis/rest_api/internal/common"
	auth_middleware "mkk_basis/rest_api/internal/components/http/middlewares"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

var taskErrorStatuses = []rest_common.ErrorStatus{
	{Error: tasks_service.ErrTaskTitleRequired, Status: http.StatusBadRequest},
	{Error: tasks_service.ErrInvalidTaskStatus, Status: http.StatusBadRequest},
	{Error: tasks_service.ErrNoTaskChanges, Status: http.StatusBadRequest},
	{Error: tasks_service.ErrAssigneeNotTeamMember, Status: http.StatusBadRequest},
	{Error: tasks_service.ErrTeamMembershipRequired, Status: http.StatusForbidden},
	{Error: tasks_service.ErrInsufficientPermission, Status: http.StatusForbidden},
	{Error: tasks_service.ErrTaskNotFound, Status: http.StatusNotFound},
}

func AddRoutes(r *gin.RouterGroup) {
	router := r.Group("/tasks")
	router.Use(auth_middleware.AuthMiddleware())
	{
		router.POST("", createTaskRoute)
		router.GET("", getTasksRoute)
		router.PUT("/:id", updateTaskRoute)
		router.GET("/:id/history", getTaskHistoryRoute)
	}
}

// Create task
//
//	@Summary		Create task
//	@Description	Создать задачу в команде
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			request	body		tasks_entities.CreateTaskRequest	true	"Task data"
//	@Success		201		{object}	common.APIResponse{result=tasks_entities.TaskResponse}
//	@Router			/api/v1/tasks [post]
func createTaskRoute(c *gin.Context) {
	userID, ok := rest_common.CurrentUserID(c)
	if !ok {
		return
	}

	var request tasks_entities.CreateTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	task, err := tasks_handler.CreateTask(c.Request.Context(), userID, &request)
	if err != nil {
		rest_common.WriteError(c, err, taskErrorStatuses...)
		return
	}

	c.JSON(http.StatusCreated, common.ResultResponseNoErr(task))
}

// Get tasks
//
//	@Summary		Get tasks
//	@Description	Получить задачи команды с фильтрацией и пагинацией
//	@Tags			tasks
//	@Produce		json
//	@Param			query	query		tasks_entities.TaskFilterRequest	true	"Filter parameters"
//	@Success		200		{object}	common.APIResponse{result=[]tasks_entities.TaskResponse}
//	@Router			/api/v1/tasks [get]
func getTasksRoute(c *gin.Context) {
	userID, ok := rest_common.CurrentUserID(c)
	if !ok {
		return
	}

	var params tasks_entities.TaskFilterRequest
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	result, err := tasks_handler.GetTasks(c.Request.Context(), userID, &params)
	if err != nil {
		rest_common.WriteError(c, err, taskErrorStatuses...)
		return
	}

	c.Header("Content-Range", strconv.FormatInt(result.ContentRange, 10))
	c.JSON(http.StatusOK, common.ResultResponseNoErr(result.Tasks))
}

// Update task
//
//	@Summary		Update task
//	@Description	Обновить задачу
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int									true	"Task ID"
//	@Param			request	body		tasks_entities.UpdateTaskRequest	true	"Task data"
//	@Success		200		{object}	common.APIResponse{result=tasks_entities.TaskResponse}
//	@Router			/api/v1/tasks/{id} [put]
func updateTaskRoute(c *gin.Context) {
	userID, ok := rest_common.CurrentUserID(c)
	if !ok {
		return
	}

	taskID, ok := rest_common.Uint64PathParam(c, "id", "invalid task id")
	if !ok {
		return
	}

	var request tasks_entities.UpdateTaskRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(err))
		return
	}

	task, err := tasks_handler.UpdateTask(c.Request.Context(), taskID, userID, &request)
	if err != nil {
		rest_common.WriteError(c, err, taskErrorStatuses...)
		return
	}

	c.JSON(http.StatusOK, common.ResultResponseNoErr(task))
}

// Get task history
//
//	@Summary		Get task history
//	@Description	Получить историю изменений задачи
//	@Tags			tasks
//	@Produce		json
//	@Param			id	path		int	true	"Task ID"
//	@Success		200	{object}	common.APIResponse{result=[]tasks_entities.TaskHistoryResponse}
//	@Router			/api/v1/tasks/{id}/history [get]
func getTaskHistoryRoute(c *gin.Context) {
	userID, ok := rest_common.CurrentUserID(c)
	if !ok {
		return
	}

	taskID, ok := rest_common.Uint64PathParam(c, "id", "invalid task id")
	if !ok {
		return
	}

	history, err := tasks_handler.GetTaskHistory(c.Request.Context(), taskID, userID)
	if err != nil {
		rest_common.WriteError(c, err, taskErrorStatuses...)
		return
	}

	c.JSON(http.StatusOK, common.ResultResponseNoErr(history))
}
