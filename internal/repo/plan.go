package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

// scanPlanRow scans a plan row, handling DATE -> string conversion for start_date and end_date.
func scanPlanRow(scan func(dest ...any) error) (*model.Plan, error) {
	var p model.Plan
	var startDate time.Time
	var endDate *time.Time
	err := scan(&p.ID, &p.UserID, &p.Title, &p.Description, &p.Visibility,
		&startDate, &endDate, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.StartDate = startDate.Format("2006-01-02")
	if endDate != nil {
		s := endDate.Format("2006-01-02")
		p.EndDate = &s
	}
	return &p, nil
}

type PlanRepo struct {
	pool *pgxpool.Pool
}

func NewPlanRepo(pool *pgxpool.Pool) *PlanRepo {
	return &PlanRepo{pool: pool}
}

func (r *PlanRepo) Create(ctx context.Context, userID uuid.UUID, params model.CreatePlanParams) (*model.Plan, error) {
	visibility := params.Visibility
	if visibility == "" {
		visibility = "private"
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx,
		`INSERT INTO plans (user_id, title, description, visibility, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, title, description, visibility, start_date, end_date, created_at, updated_at`,
		userID, params.Title, params.Description, visibility, params.StartDate, params.EndDate,
	)
	plan, err := scanPlanRow(row.Scan)
	if err != nil {
		return nil, err
	}

	// Auto-add creator as owner
	_, err = tx.Exec(ctx,
		`INSERT INTO plan_members (plan_id, user_id, role) VALUES ($1, $2, 'owner')`,
		plan.ID, userID,
	)
	if err != nil {
		return nil, err
	}

	return plan, tx.Commit(ctx)
}

func (r *PlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Plan, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, description, visibility, start_date, end_date, created_at, updated_at
		 FROM plans WHERE id = $1`, id,
	)
	plan, err := scanPlanRow(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return plan, err
}

func (r *PlanRepo) ListByUser(ctx context.Context, userID uuid.UUID, params model.PaginationParams) ([]model.Plan, int, error) {
	params.Normalize()

	countQuery := `SELECT COUNT(*) FROM plans p
		JOIN plan_members pm ON p.id = pm.plan_id
		WHERE pm.user_id = $1`

	var total int
	err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.user_id, p.title, p.description, p.visibility, p.start_date, p.end_date, p.created_at, p.updated_at
		 FROM plans p
		 JOIN plan_members pm ON p.id = pm.plan_id
		 WHERE pm.user_id = $1 ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`,
		userID, params.PageSize, params.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var plans []model.Plan
	for rows.Next() {
		p, err := scanPlanRow(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		plans = append(plans, *p)
	}
	return plans, total, nil
}

func (r *PlanRepo) Update(ctx context.Context, id uuid.UUID, params model.UpdatePlanParams) (*model.Plan, error) {
	sets := []string{}
	args := []any{}
	argIdx := 1

	if params.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *params.Title)
		argIdx++
	}
	if params.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *params.Description)
		argIdx++
	}
	if params.StartDate != nil {
		sets = append(sets, fmt.Sprintf("start_date = $%d", argIdx))
		args = append(args, *params.StartDate)
		argIdx++
	}
	if params.EndDate != nil {
		sets = append(sets, fmt.Sprintf("end_date = $%d", argIdx))
		args = append(args, *params.EndDate)
		argIdx++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE plans SET %s WHERE id = $%d
		 RETURNING id, user_id, title, description, visibility, start_date, end_date, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx,
	)

	row := r.pool.QueryRow(ctx, query, args...)
	plan, err := scanPlanRow(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return plan, err
}

func (r *PlanRepo) UpdateVisibility(ctx context.Context, id uuid.UUID, visibility string) (*model.Plan, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE plans SET visibility = $1, updated_at = NOW() WHERE id = $2
		 RETURNING id, user_id, title, description, visibility, start_date, end_date, created_at, updated_at`,
		visibility, id,
	)
	plan, err := scanPlanRow(row.Scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return plan, err
}

func (r *PlanRepo) AddMember(ctx context.Context, planID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO plan_members (plan_id, user_id, role) VALUES ($1, $2, 'member')
		 ON CONFLICT (plan_id, user_id) DO NOTHING`,
		planID, userID,
	)
	return err
}

func (r *PlanRepo) ListMembers(ctx context.Context, planID uuid.UUID) ([]model.PlanMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT pm.plan_id, pm.user_id, pm.role, pm.joined_at, u.nickname
		 FROM plan_members pm JOIN users u ON pm.user_id = u.id
		 WHERE pm.plan_id = $1 ORDER BY pm.joined_at`,
		planID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.PlanMember
	for rows.Next() {
		var m model.PlanMember
		if err := rows.Scan(&m.PlanID, &m.UserID, &m.Role, &m.JoinedAt, &m.Nickname); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (r *PlanRepo) IsMember(ctx context.Context, planID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM plan_members WHERE plan_id = $1 AND user_id = $2)`,
		planID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *PlanRepo) MemberCount(ctx context.Context, planID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM plan_members WHERE plan_id = $1`,
		planID,
	).Scan(&count)
	return count, err
}

func (r *PlanRepo) Delete(ctx context.Context, planID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete plan members first
	_, err = tx.Exec(ctx, `DELETE FROM plan_members WHERE plan_id = $1`, planID)
	if err != nil {
		return err
	}

	// Delete check-ins
	_, err = tx.Exec(ctx, `DELETE FROM check_ins WHERE plan_id = $1`, planID)
	if err != nil {
		return err
	}

	// Delete the plan
	result, err := tx.Exec(ctx, `DELETE FROM plans WHERE id = $1`, planID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return tx.Commit(ctx)
}
