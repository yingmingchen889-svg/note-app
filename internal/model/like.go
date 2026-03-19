package model

import (
	"time"

	"github.com/google/uuid"
)

type Like struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	TargetType string    `json:"target_type"`
	TargetID   uuid.UUID `json:"target_id"`
	CreatedAt  time.Time `json:"created_at"`
}
