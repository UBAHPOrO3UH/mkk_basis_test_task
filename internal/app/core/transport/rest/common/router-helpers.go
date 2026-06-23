package rest_common

import (
	"errors"
	"mkk_basis/rest_api/internal/common"
	auth_middleware "mkk_basis/rest_api/internal/components/http/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorStatus struct {
	Error  error
	Status int
}

func CurrentUserID(c *gin.Context) (uint64, bool) {
	claims, ok := auth_middleware.ClaimsFromContext(c.Request.Context())
	if !ok {
		writeUnauthorized(c)
		return 0, false
	}

	userID, err := claims.UserID()
	if err != nil {
		writeUnauthorized(c)
		return 0, false
	}

	return userID, true
}

func WriteError(c *gin.Context, err error, statuses ...ErrorStatus) {
	status := http.StatusInternalServerError
	for _, candidate := range statuses {
		if errors.Is(err, candidate.Error) {
			status = candidate.Status
			break
		}
	}

	c.JSON(status, common.ErrorResponse(err))
}

func writeUnauthorized(c *gin.Context) {
	c.JSON(
		http.StatusUnauthorized,
		common.ErrorResponse(errors.New("authentication required")),
	)
}
