package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

type CommentRepo struct {
	pool *pgxpool.Pool
}

func NewCommentRepo(pool *pgxpool.Pool) *CommentRepo {
	return &CommentRepo{pool: pool}
}

func (r *CommentRepo) Create(ctx context.Context, userID uuid.UUID, targetType string, targetID uuid.UUID, params model.CreateCommentParams) (*model.Comment, error) {
	var c model.Comment
	err := r.pool.QueryRow(ctx,
		`INSERT INTO comments (user_id, target_type, target_id, parent_id, content)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, target_type, target_id, parent_id, content, created_at, updated_at`,
		userID, targetType, targetID, params.ParentID, params.Content,
	).Scan(&c.ID, &c.UserID, &c.TargetType, &c.TargetID, &c.ParentID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}

func (r *CommentRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	var c model.Comment
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, target_type, target_id, parent_id, content, created_at, updated_at
		 FROM comments WHERE id = $1`, id,
	).Scan(&c.ID, &c.UserID, &c.TargetType, &c.TargetID, &c.ParentID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

// ListByTarget returns top-level comments (parent_id IS NULL) with user info,
// reply count, and up to 3 earliest reply previews per comment.
func (r *CommentRepo) ListByTarget(ctx context.Context, targetType string, targetID uuid.UUID, params model.PaginationParams) ([]model.CommentWithUser, int, error) {
	params.Normalize()

	// Count top-level comments
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM comments WHERE target_type = $1 AND target_id = $2 AND parent_id IS NULL`,
		targetType, targetID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch top-level comments with user info
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.content, c.parent_id, c.created_at,
		        u.id, u.nickname, u.avatar_url,
		        (SELECT COUNT(*) FROM comments r WHERE r.parent_id = c.id) AS reply_count
		 FROM comments c
		 JOIN users u ON c.user_id = u.id
		 WHERE c.target_type = $1 AND c.target_id = $2 AND c.parent_id IS NULL
		 ORDER BY c.created_at ASC
		 LIMIT $3 OFFSET $4`,
		targetType, targetID, params.PageSize, params.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []model.CommentWithUser
	for rows.Next() {
		var cw model.CommentWithUser
		if err := rows.Scan(&cw.ID, &cw.Content, &cw.ParentID, &cw.CreatedAt,
			&cw.User.ID, &cw.User.Nickname, &cw.User.AvatarURL, &cw.ReplyCount); err != nil {
			return nil, 0, err
		}
		comments = append(comments, cw)
	}

	// Fetch reply previews (earliest 3) for each top-level comment
	for i := range comments {
		previews, err := r.fetchReplyPreviews(ctx, comments[i].ID, 3)
		if err != nil {
			return nil, 0, err
		}
		comments[i].Replies = previews
	}

	return comments, total, nil
}

func (r *CommentRepo) fetchReplyPreviews(ctx context.Context, parentID uuid.UUID, limit int) ([]model.CommentWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.content, c.parent_id, c.created_at,
		        u.id, u.nickname, u.avatar_url
		 FROM comments c
		 JOIN users u ON c.user_id = u.id
		 WHERE c.parent_id = $1
		 ORDER BY c.created_at ASC
		 LIMIT $2`,
		parentID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var replies []model.CommentWithUser
	for rows.Next() {
		var cw model.CommentWithUser
		if err := rows.Scan(&cw.ID, &cw.Content, &cw.ParentID, &cw.CreatedAt,
			&cw.User.ID, &cw.User.Nickname, &cw.User.AvatarURL); err != nil {
			return nil, err
		}
		replies = append(replies, cw)
	}
	return replies, nil
}

// ListReplies returns all replies to a given comment, paginated.
func (r *CommentRepo) ListReplies(ctx context.Context, parentID uuid.UUID, params model.PaginationParams) ([]model.CommentWithUser, int, error) {
	params.Normalize()

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM comments WHERE parent_id = $1`, parentID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.content, c.parent_id, c.created_at,
		        u.id, u.nickname, u.avatar_url
		 FROM comments c
		 JOIN users u ON c.user_id = u.id
		 WHERE c.parent_id = $1
		 ORDER BY c.created_at ASC
		 LIMIT $2 OFFSET $3`,
		parentID, params.PageSize, params.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var replies []model.CommentWithUser
	for rows.Next() {
		var cw model.CommentWithUser
		if err := rows.Scan(&cw.ID, &cw.Content, &cw.ParentID, &cw.CreatedAt,
			&cw.User.ID, &cw.User.Nickname, &cw.User.AvatarURL); err != nil {
			return nil, 0, err
		}
		replies = append(replies, cw)
	}
	return replies, total, nil
}

func (r *CommentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, "DELETE FROM comments WHERE id = $1", id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CommentRepo) CountByTarget(ctx context.Context, targetType string, targetID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM comments WHERE target_type = $1 AND target_id = $2`,
		targetType, targetID,
	).Scan(&count)
	return count, err
}
