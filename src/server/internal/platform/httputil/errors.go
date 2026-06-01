package httputil

import (
	"github.com/gin-gonic/gin"
)

// ErrorResponse represents the standard API error structure.
// @Name ErrorResponse
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code,omitempty"`
}

func WriteError(c *gin.Context, httpStatus int, message string) {
	c.JSON(httpStatus, ErrorResponse{
		Error: message,
		Code:  httpStatus,
	})
}

func AbortWithError(c *gin.Context, httpStatus int, message string) {
	c.AbortWithStatusJSON(httpStatus, ErrorResponse{
		Error: message,
		Code:  httpStatus,
	})
}
