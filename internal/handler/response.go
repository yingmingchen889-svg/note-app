package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/model"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func RespondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

func RespondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

func RespondPaginated(c *gin.Context, data any, total int, params model.PaginationParams) {
	c.JSON(http.StatusOK, model.PaginatedResponse{
		Data:     data,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	})
}

func RespondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{Code: code, Message: message})
}

func RespondBadRequest(c *gin.Context, message string) {
	RespondError(c, http.StatusBadRequest, "INVALID_INPUT", message)
}

func RespondUnauthorized(c *gin.Context) {
	RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

func RespondForbidden(c *gin.Context) {
	RespondError(c, http.StatusForbidden, "FORBIDDEN", "permission denied")
}

func RespondNotFound(c *gin.Context) {
	RespondError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
}

func RespondConflict(c *gin.Context, message string) {
	RespondError(c, http.StatusConflict, "CONFLICT", message)
}

func RespondInternalError(c *gin.Context) {
	RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
