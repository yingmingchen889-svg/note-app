package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/service"
)

type CheckInHandler struct {
	checkInService *service.CheckInService
}

func NewCheckInHandler(checkInService *service.CheckInService) *CheckInHandler {
	return &CheckInHandler{checkInService: checkInService}
}

func (h *CheckInHandler) CheckIn(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}

	var params model.UpsertCheckInParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	ci, err := h.checkInService.CheckIn(c.Request.Context(), getUserID(c), planID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondCreated(c, ci)
}

func (h *CheckInHandler) ListByPlan(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}

	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	checkins, total, err := h.checkInService.ListByPlan(c.Request.Context(), planID, params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, checkins, total, params)
}

func (h *CheckInHandler) Calendar(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")
	if startDate == "" || endDate == "" {
		RespondBadRequest(c, "start_date and end_date are required (YYYY-MM-DD)")
		return
	}

	entries, err := h.checkInService.Calendar(c.Request.Context(), getUserID(c), startDate, endDate)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondOK(c, entries)
}
