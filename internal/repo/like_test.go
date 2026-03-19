package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestLikeRepo_CreateAndExists(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	likeRepo := NewLikeRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "liker@test.com", Password: "x", Nickname: "Liker",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Likeable"})

	// Create like
	err := likeRepo.Create(ctx, user.ID, "note", note.ID)
	require.NoError(t, err)

	// Check exists
	exists, err := likeRepo.Exists(ctx, user.ID, "note", note.ID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Count
	count, err := likeRepo.CountByTarget(ctx, "note", note.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestLikeRepo_CreateIdempotent(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	likeRepo := NewLikeRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "idempotent@test.com", Password: "x", Nickname: "Idem",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Test"})

	_ = likeRepo.Create(ctx, user.ID, "note", note.ID)
	err := likeRepo.Create(ctx, user.ID, "note", note.ID) // duplicate
	require.NoError(t, err) // should not error

	count, _ := likeRepo.CountByTarget(ctx, "note", note.ID)
	assert.Equal(t, 1, count) // still 1
}

func TestLikeRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	likeRepo := NewLikeRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "unliker@test.com", Password: "x", Nickname: "Unliker",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Unlikeable"})

	_ = likeRepo.Create(ctx, user.ID, "note", note.ID)
	err := likeRepo.Delete(ctx, user.ID, "note", note.ID)
	require.NoError(t, err)

	exists, _ := likeRepo.Exists(ctx, user.ID, "note", note.ID)
	assert.False(t, exists)
}
