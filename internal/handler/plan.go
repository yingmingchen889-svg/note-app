package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/service"
)

type PlanHandler struct {
	planService *service.PlanService
}

func NewPlanHandler(planService *service.PlanService) *PlanHandler {
	return &PlanHandler{planService: planService}
}

func (h *PlanHandler) Create(c *gin.Context) {
	var params model.CreatePlanParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	plan, err := h.planService.Create(c.Request.Context(), getUserID(c), params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondCreated(c, plan)
}

func (h *PlanHandler) Get(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	plan, err := h.planService.GetByID(c.Request.Context(), getUserID(c), planID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, plan)
}

func (h *PlanHandler) List(c *gin.Context) {
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()
	plans, total, err := h.planService.List(c.Request.Context(), getUserID(c), params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, plans, total, params)
}

func (h *PlanHandler) Update(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	var params model.UpdatePlanParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	plan, err := h.planService.Update(c.Request.Context(), getUserID(c), planID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, plan)
}

func (h *PlanHandler) Share(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	plan, err := h.planService.Share(c.Request.Context(), getUserID(c), planID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, plan)
}

func (h *PlanHandler) Join(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	if err := h.planService.Join(c.Request.Context(), getUserID(c), planID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "joined"})
}

func (h *PlanHandler) Members(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid plan id")
		return
	}
	members, err := h.planService.ListMembers(c.Request.Context(), planID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, members)
}
