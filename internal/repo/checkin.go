package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

type CheckInRepo struct {
	pool *pgxpool.Pool
}

func NewCheckInRepo(pool *pgxpool.Pool) *CheckInRepo {
	return &CheckInRepo{pool: pool}
}

// scanCheckIn scans a check_in row, handling DATE -> string conversion for checked_date.
func scanCheckIn(scan func(dest ...any) error) (*model.CheckIn, error) {
	var ci model.CheckIn
	var checkedDate time.Time
	err := scan(&ci.ID, &ci.PlanID, &ci.UserID, &ci.Content, &ci.Media, &checkedDate, &ci.CheckedAt)
	if err != nil {
		return nil, err
	}
	ci.CheckedDate = checkedDate.Format("2006-01-02")
	return &ci, nil
}

func (r *CheckInRepo) Upsert(ctx context.Context, planID, userID uuid.UUID, date string, params model.UpsertCheckInParams) (*model.CheckIn, bool, error) {
	media := params.Media
	if media == nil {
		media = json.RawMessage(`[]`)
	}

	var isNew bool
	row := r.pool.QueryRow(ctx,
		`WITH upsert AS (
			INSERT INTO check_ins (plan_id, user_id, content, media, checked_date)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (plan_id, user_id, checked_date)
			DO UPDATE SET content = EXCLUDED.content, media = EXCLUDED.media, checked_at = NOW()
			RETURNING *, (xmax = 0) AS is_new
		) SELECT id, plan_id, user_id, content, media, checked_date, checked_at, is_new FROM upsert`,
		planID, userID, params.Content, media, date,
	)

	var ci model.CheckIn
	var checkedDate time.Time
	err := row.Scan(&ci.ID, &ci.PlanID, &ci.UserID, &ci.Content, &ci.Media, &checkedDate, &ci.CheckedAt, &isNew)
	if err != nil {
		return nil, false, err
	}
	ci.CheckedDate = checkedDate.Format("2006-01-02")
	return &ci, isNew, nil
}

func (r *CheckInRepo) ListByPlan(ctx context.Context, planID uuid.UUID, params model.PaginationParams) ([]model.CheckIn, int, error) {
	params.Normalize()

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM check_ins WHERE plan_id = $1", planID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, plan_id, user_id, content, media, checked_date, checked_at
		 FROM check_ins WHERE plan_id = $1 ORDER BY checked_date DESC LIMIT $2 OFFSET $3`,
		planID, params.PageSize, params.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var checkins []model.CheckIn
	for rows.Next() {
		ci, err := scanCheckIn(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		checkins = append(checkins, *ci)
	}
	return checkins, total, nil
}

func (r *CheckInRepo) Calendar(ctx context.Context, userID uuid.UUID, startDate, endDate string) ([]model.CalendarEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ci.checked_date, ci.plan_id, p.title
		 FROM check_ins ci JOIN plans p ON ci.plan_id = p.id
		 WHERE ci.user_id = $1 AND ci.checked_date >= $2 AND ci.checked_date <= $3
		 ORDER BY ci.checked_date`,
		userID, startDate, endDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.CalendarEntry
	for rows.Next() {
		var e model.CalendarEntry
		var checkedDate time.Time
		if err := rows.Scan(&checkedDate, &e.PlanID, &e.PlanTitle); err != nil {
			return nil, err
		}
		e.Date = checkedDate.Format("2006-01-02")
		entries = append(entries, e)
	}
	return entries, nil
}

func (r *CheckInRepo) CurrentStreak(ctx context.Context, planID, userID uuid.UUID, today string) (int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT checked_date FROM check_ins
		 WHERE plan_id = $1 AND user_id = $2 AND checked_date <= $3
		 ORDER BY checked_date DESC`,
		planID, userID, today,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	streak := 0
	todayTime, _ := time.Parse("2006-01-02", today)
	var prevDate time.Time

	for rows.Next() {
		var rawDate time.Time
		if err := rows.Scan(&rawDate); err != nil {
			return 0, err
		}
		// Normalize to UTC date-only to avoid timezone issues
		date := time.Date(rawDate.Year(), rawDate.Month(), rawDate.Day(), 0, 0, 0, 0, time.UTC)

		if streak == 0 {
			todayNorm := time.Date(todayTime.Year(), todayTime.Month(), todayTime.Day(), 0, 0, 0, 0, time.UTC)
			if date.Equal(todayNorm) {
				streak = 1
				prevDate = date
				continue
			}
			break
		}

		// Check if this date is exactly 1 day before prevDate
		expected := prevDate.AddDate(0, 0, -1)
		if !date.Equal(expected) {
			break
		}
		streak++
		prevDate = date
	}
	return streak, nil
}

// GetByID retrieves a single check-in by ID.
func (r *CheckInRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.CheckIn, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, plan_id, user_id, content, media, checked_date, checked_at
		 FROM check_ins WHERE id = $1`, id,
	)
	ci, err := scanCheckIn(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return ci, err
}
