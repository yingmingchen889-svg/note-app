package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type GrowthReport struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	PeriodType  string          `json:"period_type"`
	PeriodStart string          `json:"period_start"`
	Stats       json.RawMessage `json:"stats"`
	GeneratedAt time.Time       `json:"generated_at"`
}

type GenerateReportParams struct {
	PeriodType  string `json:"period_type" binding:"required,oneof=monthly quarterly yearly"`
	PeriodStart string `json:"period_start" binding:"required"`
}

// GrowthStats is the structure stored as JSONB in the stats column.
type GrowthStats struct {
	TotalCheckIns  int            `json:"total_check_ins"`
	TotalNotes     int            `json:"total_notes"`
	LongestStreak  int            `json:"longest_streak"`
	PlanStats      []PlanStat     `json:"plan_stats"`
	TopPlans       []TopPlan      `json:"top_plans"`
	DailyCheckIns  map[string]int `json:"daily_check_ins"`
	WeeklyNotes    map[string]int `json:"weekly_notes"`
}

type PlanStat struct {
	PlanID         uuid.UUID `json:"plan_id"`
	PlanTitle      string    `json:"plan_title"`
	TotalDays      int       `json:"total_days"`
	CheckedDays    int       `json:"checked_days"`
	CompletionRate float64   `json:"completion_rate"`
}

type TopPlan struct {
	PlanID    uuid.UUID `json:"plan_id"`
	PlanTitle string    `json:"plan_title"`
	CheckIns  int       `json:"check_ins"`
}
