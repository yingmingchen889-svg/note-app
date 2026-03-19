package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestExploreRepo_ListPublicNotes(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	exploreRepo := NewExploreRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "explorer@test.com", Password: "x", Nickname: "Explorer",
	}, "$2a$10$dummyhash")

	// Create private + public notes
	_, _ = noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Private", Visibility: "private"})
	pub, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Public"})
	vis := "public"
	_, _ = noteRepo.Update(ctx, pub.ID, model.UpdateNoteParams{Visibility: &vis})

	notes, total, err := exploreRepo.ListPublicNotes(ctx, uuid.Nil, model.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, notes, 1)
	assert.Equal(t, "Public", notes[0]["title"])
}
