package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

type ExploreRepo struct {
	pool *pgxpool.Pool
}

func NewExploreRepo(pool *pgxpool.Pool) *ExploreRepo {
	return &ExploreRepo{pool: pool}
}

// ListPublicNotes returns public notes with author info, like/comment counts, is_liked.
func (r *ExploreRepo) ListPublicNotes(ctx context.Context, currentUserID uuid.UUID, params model.PaginationParams) ([]map[string]any, int, error) {
	params.Normalize()

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notes WHERE visibility = 'public' AND is_draft = false`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	isLikedExpr := "false"
	args := []any{}
	argIdx := 1
	if currentUserID != uuid.Nil {
		isLikedExpr = fmt.Sprintf(
			"EXISTS(SELECT 1 FROM likes WHERE user_id = $%d AND target_type = 'note' AND target_id = n.id)", argIdx)
		args = append(args, currentUserID)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT n.id, n.title, n.content, n.tags, n.created_at,
		       u.id, u.nickname, u.avatar_url,
		       (SELECT COUNT(*) FROM likes WHERE target_type = 'note' AND target_id = n.id) AS like_count,
		       (SELECT COUNT(*) FROM comments WHERE target_type = 'note' AND target_id = n.id) AS comment_count,
		       %s AS is_liked
		FROM notes n
		JOIN users u ON n.user_id = u.id
		WHERE n.visibility = 'public' AND n.is_draft = false
		ORDER BY n.created_at DESC
		LIMIT $%d OFFSET $%d`, isLikedExpr, argIdx, argIdx+1)
	args = append(args, params.PageSize, params.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var (
			noteID                uuid.UUID
			title, content        string
			tags                  json.RawMessage
			createdAt             any
			authorID              uuid.UUID
			nickname              string
			avatarURL             *string
			likeCount, commentCnt int
			isLiked               bool
		)
		if err := rows.Scan(&noteID, &title, &content, &tags, &createdAt,
			&authorID, &nickname, &avatarURL,
			&likeCount, &commentCnt, &isLiked); err != nil {
			return nil, 0, err
		}
		results = append(results, map[string]any{
			"id":            noteID,
			"title":         title,
			"content":       content,
			"tags":          tags,
			"created_at":    createdAt,
			"author":        map[string]any{"id": authorID, "nickname": nickname, "avatar_url": avatarURL},
			"like_count":    likeCount,
			"comment_count": commentCnt,
			"is_liked":      isLiked,
		})
	}
	return results, total, nil
}

// ListPublicPlans returns public plans with author info, like/comment counts, member count.
func (r *ExploreRepo) ListPublicPlans(ctx context.Context, currentUserID uuid.UUID, params model.PaginationParams) ([]map[string]any, int, error) {
	params.Normalize()

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM plans WHERE visibility = 'public'`,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	isLikedExpr := "false"
	args := []any{}
	argIdx := 1
	if currentUserID != uuid.Nil {
		isLikedExpr = fmt.Sprintf(
			"EXISTS(SELECT 1 FROM likes WHERE user_id = $%d AND target_type = 'plan' AND target_id = p.id)", argIdx)
		args = append(args, currentUserID)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.title, p.description, p.start_date, p.end_date, p.created_at,
		       u.id, u.nickname, u.avatar_url,
		       (SELECT COUNT(*) FROM likes WHERE target_type = 'plan' AND target_id = p.id) AS like_count,
		       (SELECT COUNT(*) FROM comments WHERE target_type = 'plan' AND target_id = p.id) AS comment_count,
		       (SELECT COUNT(*) FROM plan_members WHERE plan_id = p.id) AS member_count,
		       %s AS is_liked
		FROM plans p
		JOIN users u ON p.user_id = u.id
		WHERE p.visibility = 'public'
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d`, isLikedExpr, argIdx, argIdx+1)
	args = append(args, params.PageSize, params.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var (
			planID                           uuid.UUID
			title, description               string
			startDate, endDate               any
			createdAt                        any
			authorID                         uuid.UUID
			nickname                         string
			avatarURL                        *string
			likeCount, commentCnt, memberCnt int
			isLiked                          bool
		)
		if err := rows.Scan(&planID, &title, &description, &startDate, &endDate, &createdAt,
			&authorID, &nickname, &avatarURL,
			&likeCount, &commentCnt, &memberCnt, &isLiked); err != nil {
			return nil, 0, err
		}
		results = append(results, map[string]any{
			"id":            planID,
			"title":         title,
			"description":   description,
			"start_date":    startDate,
			"end_date":      endDate,
			"created_at":    createdAt,
			"author":        map[string]any{"id": authorID, "nickname": nickname, "avatar_url": avatarURL},
			"like_count":    likeCount,
			"comment_count": commentCnt,
			"member_count":  memberCnt,
			"is_liked":      isLiked,
		})
	}
	return results, total, nil
}
