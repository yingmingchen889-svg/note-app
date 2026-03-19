package model

import (
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Visibility  string    `json:"visibility"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreatePlanParams struct {
	Title       string  `json:"title" binding:"required,max=500"`
	Description string  `json:"description"`
	Visibility  string  `json:"visibility" binding:"omitempty,oneof=private public"`
	StartDate   string  `json:"start_date" binding:"required"`
	EndDate     *string `json:"end_date"`
}

type UpdatePlanParams struct {
	Title       *string `json:"title" binding:"omitempty,max=500"`
	Description *string `json:"description"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
}
