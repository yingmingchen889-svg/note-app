package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID         uuid.UUID       `json:"id"`
	UserID     uuid.UUID       `json:"user_id"`
	Title      string          `json:"title"`
	Content    string          `json:"content"`
	Media      json.RawMessage `json:"media"`
	Tags       json.RawMessage `json:"tags"`
	Visibility string          `json:"visibility"`
	IsDraft    bool            `json:"is_draft"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type CreateNoteParams struct {
	Title      string          `json:"title" binding:"required,max=500"`
	Content    string          `json:"content"`
	Media      json.RawMessage `json:"media"`
	Tags       json.RawMessage `json:"tags"`
	Visibility string          `json:"visibility" binding:"omitempty,oneof=private public"`
	IsDraft    bool            `json:"is_draft"`
}

type UpdateNoteParams struct {
	Title      *string          `json:"title" binding:"omitempty,max=500"`
	Content    *string          `json:"content"`
	Media      *json.RawMessage `json:"media"`
	Tags       *json.RawMessage `json:"tags"`
	Visibility *string          `json:"visibility" binding:"omitempty,oneof=private public"`
	IsDraft    *bool            `json:"is_draft"`
}

type NoteListParams struct {
	Tag string `form:"tag"`
	PaginationParams
}
