package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/service"
)

type SocialHandler struct {
	socialService *service.SocialService
}

func NewSocialHandler(socialService *service.SocialService) *SocialHandler {
	return &SocialHandler{socialService: socialService}
}

func parseTarget(c *gin.Context) (dbType string, id uuid.UUID, ok bool) {
	urlType := c.Param("target_type")
	dbType, valid := model.TargetTypeFromURL(urlType)
	if !valid {
		RespondBadRequest(c, "invalid target type, must be notes/plans/checkins")
		return "", uuid.Nil, false
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid target id")
		return "", uuid.Nil, false
	}
	return dbType, id, true
}

func (h *SocialHandler) Like(c *gin.Context) {
	targetType, targetID, ok := parseTarget(c)
	if !ok {
		return
	}
	if err := h.socialService.Like(c.Request.Context(), getUserID(c), targetType, targetID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "liked"})
}

func (h *SocialHandler) Unlike(c *gin.Context) {
	targetType, targetID, ok := parseTarget(c)
	if !ok {
		return
	}
	if err := h.socialService.Unlike(c.Request.Context(), getUserID(c), targetType, targetID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "unliked"})
}

func (h *SocialHandler) GetComments(c *gin.Context) {
	targetType, targetID, ok := parseTarget(c)
	if !ok {
		return
	}
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	comments, total, err := h.socialService.GetComments(c.Request.Context(), targetType, targetID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondPaginated(c, comments, total, params)
}

func (h *SocialHandler) CreateComment(c *gin.Context) {
	targetType, targetID, ok := parseTarget(c)
	if !ok {
		return
	}
	var params model.CreateCommentParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	comment, err := h.socialService.Comment(c.Request.Context(), getUserID(c), targetType, targetID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondCreated(c, comment)
}

func (h *SocialHandler) DeleteComment(c *gin.Context) {
	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid comment id")
		return
	}
	if err := h.socialService.DeleteComment(c.Request.Context(), getUserID(c), commentID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "deleted"})
}

func (h *SocialHandler) GetReplies(c *gin.Context) {
	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid comment id")
		return
	}
	var params model.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	replies, total, err := h.socialService.GetReplies(c.Request.Context(), commentID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondPaginated(c, replies, total, params)
}
