package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/user/note-app/internal/storage"
)

type UploadHandler struct {
	minioClient *storage.MinIOClient
}

func NewUploadHandler(minioClient *storage.MinIOClient) *UploadHandler {
	return &UploadHandler{minioClient: minioClient}
}

type PresignRequest struct {
	ContentType string `json:"content_type" binding:"required"`
}

func (h *UploadHandler) Presign(c *gin.Context) {
	var req PresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	result, err := h.minioClient.Presign(c.Request.Context(), req.ContentType)
	if err != nil {
		RespondInternalError(c)
		return
	}
	RespondOK(c, result)
}

type ConfirmRequest struct {
	ObjectKey string `json:"object_key" binding:"required"`
}

func (h *UploadHandler) Confirm(c *gin.Context) {
	var req ConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, err.Error())
		return
	}

	objectURL := h.minioClient.ObjectURL(req.ObjectKey)
	RespondOK(c, gin.H{
		"object_key": req.ObjectKey,
		"url":        objectURL,
	})
}
