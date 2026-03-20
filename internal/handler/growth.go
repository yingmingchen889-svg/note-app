package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/service"
)

type GrowthHandler struct {
	growthService *service.GrowthService
}

func NewGrowthHandler(growthService *service.GrowthService) *GrowthHandler {
	return &GrowthHandler{growthService: growthService}
}

func (h *GrowthHandler) Generate(c *gin.Context) {
	var params model.GenerateReportParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	report, err := h.growthService.Generate(c.Request.Context(), getUserID(c), params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondCreated(c, report)
}

func (h *GrowthHandler) List(c *gin.Context) {
	reports, err := h.growthService.List(c.Request.Context(), getUserID(c))
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondOK(c, reports)
}
