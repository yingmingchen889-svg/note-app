package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CheckIn struct {
	ID          uuid.UUID       `json:"id"`
	PlanID      uuid.UUID       `json:"plan_id"`
	UserID      uuid.UUID       `json:"user_id"`
	Content     string          `json:"content"`
	Media       json.RawMessage `json:"media"`
	CheckedDate string          `json:"checked_date"`
	CheckedAt   time.Time       `json:"checked_at"`
}

type UpsertCheckInParams struct {
	Content string          `json:"content"`
	Media   json.RawMessage `json:"media"`
}

type CalendarEntry struct {
	Date      string    `json:"date"`
	PlanID    uuid.UUID `json:"plan_id"`
	PlanTitle string    `json:"plan_title"`
}

type LeaderboardEntry struct {
	Rank         int       `json:"rank"`
	User         UserBrief `json:"user"`
	CheckInCount int       `json:"check_in_count"`
	Streak       int       `json:"streak"`
}
