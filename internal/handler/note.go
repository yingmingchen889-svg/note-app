package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
	"github.com/user/note-app/internal/service"
)

type NoteHandler struct {
	noteService   *service.NoteService
	socialService *service.SocialService
}

func NewNoteHandler(noteService *service.NoteService, socialService *service.SocialService) *NoteHandler {
	return &NoteHandler{noteService: noteService, socialService: socialService}
}

func getUserID(c *gin.Context) uuid.UUID {
	return c.MustGet("user_id").(uuid.UUID)
}

func (h *NoteHandler) Create(c *gin.Context) {
	var params model.CreateNoteParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	note, err := h.noteService.Create(c.Request.Context(), getUserID(c), params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondCreated(c, note)
}

func (h *NoteHandler) Get(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid note id")
		return
	}

	note, err := h.noteService.GetByID(c.Request.Context(), getUserID(c), noteID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// For public notes, include social counts
	if note.Visibility == "public" && h.socialService != nil {
		likeCount, commentCount, isLiked, _ := h.socialService.GetSocialCounts(
			c.Request.Context(), getUserID(c), "note", noteID)
		RespondOK(c, gin.H{
			"note":          note,
			"like_count":    likeCount,
			"comment_count": commentCount,
			"is_liked":      isLiked,
		})
		return
	}
	RespondOK(c, note)
}

func (h *NoteHandler) List(c *gin.Context) {
	var params model.NoteListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}
	params.Normalize()

	notes, total, err := h.noteService.List(c.Request.Context(), getUserID(c), params)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondPaginated(c, notes, total, params.PaginationParams)
}

func (h *NoteHandler) Update(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid note id")
		return
	}

	var params model.UpdateNoteParams
	if err := c.ShouldBindJSON(&params); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	note, err := h.noteService.Update(c.Request.Context(), getUserID(c), noteID, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, note)
}

func (h *NoteHandler) Delete(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid note id")
		return
	}

	if err := h.noteService.Delete(c.Request.Context(), getUserID(c), noteID); err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, gin.H{"message": "deleted"})
}

func (h *NoteHandler) Share(c *gin.Context) {
	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequest(c, "invalid note id")
		return
	}

	note, err := h.noteService.Share(c.Request.Context(), getUserID(c), noteID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	RespondOK(c, note)
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		RespondNotFound(c)
	case errors.Is(err, service.ErrForbidden):
		RespondForbidden(c)
	default:
		RespondInternalError(c)
	}
}
