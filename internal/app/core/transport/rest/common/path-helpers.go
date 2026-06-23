package rest_common

import (
	"errors"
	"mkk_basis/rest_api/internal/common"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func Uint64PathParam(c *gin.Context, name, errorMessage string) (uint64, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, common.ErrorResponse(errors.New(errorMessage)))
		return 0, false
	}

	return value, true
}
