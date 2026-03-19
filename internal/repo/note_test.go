package repo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestNoteRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "notetest@test.com", Password: "x", Nickname: "Tester",
	}, "$2a$10$dummyhash")

	note, err := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title:   "My First Note",
		Content: "Hello world",
		Tags:    json.RawMessage(`["life"]`),
	})
	require.NoError(t, err)
	assert.Equal(t, "My First Note", note.Title)
	assert.Equal(t, "private", note.Visibility)

	found, err := noteRepo.GetByID(ctx, note.ID)
	require.NoError(t, err)
	assert.Equal(t, note.ID, found.ID)
}

func TestNoteRepo_List_WithTagFilter(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "tagtest@test.com", Password: "x", Nickname: "Tagger",
	}, "$2a$10$dummyhash")

	_, _ = noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title: "Life Note", Tags: json.RawMessage(`["life"]`),
	})
	_, _ = noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title: "Work Note", Tags: json.RawMessage(`["work"]`),
	})

	// List all
	notes, total, err := noteRepo.ListByUser(ctx, user.ID, model.NoteListParams{
		PaginationParams: model.PaginationParams{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, notes, 2)

	// Filter by tag
	notes, total, err = noteRepo.ListByUser(ctx, user.ID, model.NoteListParams{
		Tag:              "life",
		PaginationParams: model.PaginationParams{Page: 1, PageSize: 20},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, notes, 1)
	assert.Equal(t, "Life Note", notes[0].Title)
}

func TestNoteRepo_Update(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "updatetest@test.com", Password: "x", Nickname: "Updater",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title: "Original",
	})

	newTitle := "Updated"
	updated, err := noteRepo.Update(ctx, note.ID, model.UpdateNoteParams{
		Title: &newTitle,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Title)
}

func TestNoteRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "deltest@test.com", Password: "x", Nickname: "Deleter",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{
		Title: "To Delete",
	})

	err := noteRepo.Delete(ctx, note.ID)
	require.NoError(t, err)

	_, err = noteRepo.GetByID(ctx, note.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}
