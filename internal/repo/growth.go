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

type GrowthRepo struct {
	pool *pgxpool.Pool
}

func NewGrowthRepo(pool *pgxpool.Pool) *GrowthRepo {
	return &GrowthRepo{pool: pool}
}

func (r *GrowthRepo) Upsert(ctx context.Context, userID uuid.UUID, periodType, periodStart string, stats json.RawMessage) (*model.GrowthReport, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO growth_reports (user_id, period_type, period_start, stats, generated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (user_id, period_type, period_start)
		 DO UPDATE SET stats = EXCLUDED.stats, generated_at = NOW()
		 RETURNING id, user_id, period_type, period_start, stats, generated_at`,
		userID, periodType, periodStart, stats,
	)
	return scanGrowthReport(row.Scan)
}

func (r *GrowthRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.GrowthReport, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, period_type, period_start, stats, generated_at
		 FROM growth_reports WHERE user_id = $1
		 ORDER BY period_start DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []model.GrowthReport
	for rows.Next() {
		rpt, err := scanGrowthReport(rows.Scan)
		if err != nil {
			return nil, err
		}
		reports = append(reports, *rpt)
	}
	return reports, nil
}

func (r *GrowthRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.GrowthReport, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, period_type, period_start, stats, generated_at
		 FROM growth_reports WHERE id = $1`, id,
	)
	rpt, err := scanGrowthReport(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rpt, err
}

func scanGrowthReport(scan func(dest ...any) error) (*model.GrowthReport, error) {
	var rpt model.GrowthReport
	var periodStart time.Time
	err := scan(&rpt.ID, &rpt.UserID, &rpt.PeriodType, &periodStart, &rpt.Stats, &rpt.GeneratedAt)
	if err != nil {
		return nil, err
	}
	rpt.PeriodStart = periodStart.Format("2006-01-02")
	return &rpt, nil
}
