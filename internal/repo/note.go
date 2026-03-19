package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/note-app/internal/model"
)

type NoteRepo struct {
	pool *pgxpool.Pool
}

func NewNoteRepo(pool *pgxpool.Pool) *NoteRepo {
	return &NoteRepo{pool: pool}
}

func (r *NoteRepo) Create(ctx context.Context, userID uuid.UUID, params model.CreateNoteParams) (*model.Note, error) {
	media := params.Media
	if media == nil {
		media = json.RawMessage(`[]`)
	}
	tags := params.Tags
	if tags == nil {
		tags = json.RawMessage(`[]`)
	}
	visibility := params.Visibility
	if visibility == "" {
		visibility = "private"
	}

	var note model.Note
	err := r.pool.QueryRow(ctx,
		`INSERT INTO notes (user_id, title, content, media, tags, visibility, is_draft)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, user_id, title, content, media, tags, visibility, is_draft, created_at, updated_at`,
		userID, params.Title, params.Content, media, tags, visibility, params.IsDraft,
	).Scan(&note.ID, &note.UserID, &note.Title, &note.Content, &note.Media,
		&note.Tags, &note.Visibility, &note.IsDraft, &note.CreatedAt, &note.UpdatedAt)
	return &note, err
}

func (r *NoteRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Note, error) {
	var note model.Note
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, title, content, media, tags, visibility, is_draft, created_at, updated_at
		 FROM notes WHERE id = $1`,
		id,
	).Scan(&note.ID, &note.UserID, &note.Title, &note.Content, &note.Media,
		&note.Tags, &note.Visibility, &note.IsDraft, &note.CreatedAt, &note.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &note, err
}

func (r *NoteRepo) ListByUser(ctx context.Context, userID uuid.UUID, params model.NoteListParams) ([]model.Note, int, error) {
	params.Normalize()

	where := "WHERE user_id = $1"
	args := []any{userID}
	argIdx := 2

	if params.Tag != "" {
		where += fmt.Sprintf(" AND tags @> $%d", argIdx)
		args = append(args, fmt.Sprintf(`["%s"]`, params.Tag))
		argIdx++
	}

	// Count
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM notes "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch
	query := fmt.Sprintf(
		`SELECT id, user_id, title, content, media, tags, visibility, is_draft, created_at, updated_at
		 FROM notes %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, params.PageSize, params.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notes []model.Note
	for rows.Next() {
		var n model.Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.Media,
			&n.Tags, &n.Visibility, &n.IsDraft, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, 0, err
		}
		notes = append(notes, n)
	}
	return notes, total, nil
}

func (r *NoteRepo) Update(ctx context.Context, id uuid.UUID, params model.UpdateNoteParams) (*model.Note, error) {
	sets := []string{}
	args := []any{}
	argIdx := 1

	if params.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *params.Title)
		argIdx++
	}
	if params.Content != nil {
		sets = append(sets, fmt.Sprintf("content = $%d", argIdx))
		args = append(args, *params.Content)
		argIdx++
	}
	if params.Media != nil {
		sets = append(sets, fmt.Sprintf("media = $%d", argIdx))
		args = append(args, *params.Media)
		argIdx++
	}
	if params.Tags != nil {
		sets = append(sets, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, *params.Tags)
		argIdx++
	}
	if params.Visibility != nil {
		sets = append(sets, fmt.Sprintf("visibility = $%d", argIdx))
		args = append(args, *params.Visibility)
		argIdx++
	}
	if params.IsDraft != nil {
		sets = append(sets, fmt.Sprintf("is_draft = $%d", argIdx))
		args = append(args, *params.IsDraft)
		argIdx++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(
		`UPDATE notes SET %s WHERE id = $%d
		 RETURNING id, user_id, title, content, media, tags, visibility, is_draft, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx,
	)

	var note model.Note
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&note.ID, &note.UserID, &note.Title, &note.Content, &note.Media,
		&note.Tags, &note.Visibility, &note.IsDraft, &note.CreatedAt, &note.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &note, err
}

func (r *NoteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	ct, err := r.pool.Exec(ctx, "DELETE FROM notes WHERE id = $1", id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *NoteRepo) UpdateVisibility(ctx context.Context, id uuid.UUID, visibility string) (*model.Note, error) {
	v := visibility
	return r.Update(ctx, id, model.UpdateNoteParams{Visibility: &v})
}
