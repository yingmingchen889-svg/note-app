package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LikeRepo struct {
	pool *pgxpool.Pool
}

func NewLikeRepo(pool *pgxpool.Pool) *LikeRepo {
	return &LikeRepo{pool: pool}
}

func (r *LikeRepo) Create(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO likes (user_id, target_type, target_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, target_type, target_id) DO NOTHING`,
		userID, targetType, targetID,
	)
	return err
}

func (r *LikeRepo) Delete(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM likes WHERE user_id = $1 AND target_type = $2 AND target_id = $3`,
		userID, targetType, targetID,
	)
	return err
}

func (r *LikeRepo) Exists(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = $1 AND target_type = $2 AND target_id = $3)`,
		userID, targetType, targetID,
	).Scan(&exists)
	return exists, err
}

func (r *LikeRepo) CountByTarget(ctx context.Context, targetType string, targetID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM likes WHERE target_type = $1 AND target_id = $2`,
		targetType, targetID,
	).Scan(&count)
	return count, err
}
