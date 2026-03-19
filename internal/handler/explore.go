package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/middleware"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type ExploreHandler struct {
	exploreRepo *repo.ExploreRepo
}

func NewExploreHandler(exploreRepo *repo.ExploreRepo) *ExploreHandler {
	return &ExploreHandler{exploreRepo: exploreRepo}
}

func (h *ExploreHandler) ListNotes(c *gin.Context) {
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	userID := middleware.GetOptionalUserID(c)
	notes, total, err := h.exploreRepo.ListPublicNotes(c.Request.Context(), userID, params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, notes, total, params)
}

func (h *ExploreHandler) ListPlans(c *gin.Context) {
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	userID := middleware.GetOptionalUserID(c)
	plans, total, err := h.exploreRepo.ListPublicPlans(c.Request.Context(), userID, params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, plans, total, params)
}
