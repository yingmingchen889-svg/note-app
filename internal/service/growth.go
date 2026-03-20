package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type GrowthService struct {
	pool       *pgxpool.Pool
	growthRepo *repo.GrowthRepo
}

func NewGrowthService(pool *pgxpool.Pool, growthRepo *repo.GrowthRepo) *GrowthService {
	return &GrowthService{pool: pool, growthRepo: growthRepo}
}

func (s *GrowthService) Generate(ctx context.Context, userID uuid.UUID, params model.GenerateReportParams) (*model.GrowthReport, error) {
	periodStart, err := time.Parse("2006-01-02", params.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("invalid period_start: %w", err)
	}

	var periodEnd time.Time
	switch params.PeriodType {
	case "monthly":
		periodEnd = periodStart.AddDate(0, 1, 0)
	case "quarterly":
		periodEnd = periodStart.AddDate(0, 3, 0)
	case "yearly":
		periodEnd = periodStart.AddDate(1, 0, 0)
	default:
		return nil, fmt.Errorf("invalid period_type: %s", params.PeriodType)
	}

	startStr := periodStart.Format("2006-01-02")
	endStr := periodEnd.Format("2006-01-02")

	// 1. Query check-ins in date range
	type checkInRow struct {
		CheckedDate time.Time
		PlanID      uuid.UUID
		PlanTitle   string
	}
	rows, err := s.pool.Query(ctx,
		`SELECT ci.checked_date, ci.plan_id, p.title
		 FROM check_ins ci
		 JOIN plan_members pm ON ci.plan_id = pm.plan_id AND pm.user_id = $1
		 JOIN plans p ON ci.plan_id = p.id
		 WHERE ci.user_id = $1 AND ci.checked_date >= $2 AND ci.checked_date < $3`,
		userID, startStr, endStr,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkIns []checkInRow
	for rows.Next() {
		var r checkInRow
		if err := rows.Scan(&r.CheckedDate, &r.PlanID, &r.PlanTitle); err != nil {
			return nil, err
		}
		checkIns = append(checkIns, r)
	}
	rows.Close()

	// 2. Query notes count and weekly breakdown
	var totalNotes int
	err = s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notes WHERE user_id = $1 AND created_at >= $2 AND created_at < $3`,
		userID, startStr, endStr,
	).Scan(&totalNotes)
	if err != nil {
		return nil, err
	}

	weeklyRows, err := s.pool.Query(ctx,
		`SELECT date_trunc('week', created_at)::date as week_start, COUNT(*)
		 FROM notes WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
		 GROUP BY week_start ORDER BY week_start`,
		userID, startStr, endStr,
	)
	if err != nil {
		return nil, err
	}
	defer weeklyRows.Close()

	weeklyNotes := make(map[string]int)
	for weeklyRows.Next() {
		var weekStart time.Time
		var count int
		if err := weeklyRows.Scan(&weekStart, &count); err != nil {
			return nil, err
		}
		weeklyNotes[weekStart.Format("2006-01-02")] = count
	}
	weeklyRows.Close()

	// 3. Calculate stats from check-in data
	totalDaysInPeriod := int(periodEnd.Sub(periodStart).Hours() / 24)

	// daily_check_ins: date -> count across all plans
	dailyCheckIns := make(map[string]int)
	// per-plan tracking
	type planInfo struct {
		title      string
		datesSet   map[string]bool
		totalCount int
	}
	planMap := make(map[uuid.UUID]*planInfo)

	// unique dates for streak calculation
	allDates := make(map[string]bool)

	for _, ci := range checkIns {
		dateStr := ci.CheckedDate.Format("2006-01-02")
		dailyCheckIns[dateStr]++
		allDates[dateStr] = true

		pi, ok := planMap[ci.PlanID]
		if !ok {
			pi = &planInfo{title: ci.PlanTitle, datesSet: make(map[string]bool)}
			planMap[ci.PlanID] = pi
		}
		if !pi.datesSet[dateStr] {
			pi.datesSet[dateStr] = true
		}
		pi.totalCount++
	}

	// longest streak
	longestStreak := calcLongestStreak(allDates)

	// plan_stats
	var planStats []model.PlanStat
	for planID, pi := range planMap {
		checkedDays := len(pi.datesSet)
		rate := 0.0
		if totalDaysInPeriod > 0 {
			rate = float64(checkedDays) / float64(totalDaysInPeriod)
		}
		planStats = append(planStats, model.PlanStat{
			PlanID:         planID,
			PlanTitle:      pi.title,
			TotalDays:      totalDaysInPeriod,
			CheckedDays:    checkedDays,
			CompletionRate: rate,
		})
	}

	// top_plans: top 3 by check-in count
	type planCount struct {
		id    uuid.UUID
		title string
		count int
	}
	var planCounts []planCount
	for planID, pi := range planMap {
		planCounts = append(planCounts, planCount{id: planID, title: pi.title, count: pi.totalCount})
	}
	sort.Slice(planCounts, func(i, j int) bool {
		return planCounts[i].count > planCounts[j].count
	})
	topN := 3
	if len(planCounts) < topN {
		topN = len(planCounts)
	}
	topPlans := make([]model.TopPlan, topN)
	for i := 0; i < topN; i++ {
		topPlans[i] = model.TopPlan{
			PlanID:    planCounts[i].id,
			PlanTitle: planCounts[i].title,
			CheckIns:  planCounts[i].count,
		}
	}

	// Build stats
	stats := model.GrowthStats{
		TotalCheckIns: len(checkIns),
		TotalNotes:    totalNotes,
		LongestStreak: longestStreak,
		PlanStats:     planStats,
		TopPlans:      topPlans,
		DailyCheckIns: dailyCheckIns,
		WeeklyNotes:   weeklyNotes,
	}

	statsJSON, err := json.Marshal(stats)
	if err != nil {
		return nil, err
	}

	return s.growthRepo.Upsert(ctx, userID, params.PeriodType, params.PeriodStart, statsJSON)
}

func (s *GrowthService) List(ctx context.Context, userID uuid.UUID) ([]model.GrowthReport, error) {
	return s.growthRepo.ListByUser(ctx, userID)
}

// calcLongestStreak finds the longest consecutive-day run from a set of date strings.
func calcLongestStreak(dates map[string]bool) int {
	if len(dates) == 0 {
		return 0
	}

	sorted := make([]time.Time, 0, len(dates))
	for d := range dates {
		t, err := time.Parse("2006-01-02", d)
		if err != nil {
			continue
		}
		sorted = append(sorted, t)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Before(sorted[j])
	})

	longest := 1
	current := 1
	for i := 1; i < len(sorted); i++ {
		diff := sorted[i].Sub(sorted[i-1]).Hours()
		if diff == 24 {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 1
		}
	}
	return longest
}
