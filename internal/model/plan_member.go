package model

import (
	"time"

	"github.com/google/uuid"
)

type PlanMember struct {
	PlanID   uuid.UUID `json:"plan_id"`
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
	Nickname string    `json:"nickname,omitempty"`
}
