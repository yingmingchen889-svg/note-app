package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestCommentRepo_CreateAndList(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "commenter@test.com", Password: "x", Nickname: "Commenter",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Commentable"})

	// Create two top-level comments
	c1, err := commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "First!"})
	require.NoError(t, err)
	assert.Equal(t, "First!", c1.Content)
	assert.Nil(t, c1.ParentID)

	_, _ = commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "Second!"})

	// List top-level comments with reply previews
	comments, total, err := commentRepo.ListByTarget(ctx, "note", note.ID, model.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, comments, 2)
}

func TestCommentRepo_ReplyAndPreview(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "replier@test.com", Password: "x", Nickname: "Replier",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Replyable"})

	parent, _ := commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "Parent"})

	// Create 4 replies
	for i := 0; i < 4; i++ {
		_, err := commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{
			Content:  "Reply " + string(rune('A'+i)),
			ParentID: &parent.ID,
		})
		require.NoError(t, err)
	}

	// List should show parent with reply_count=4 and 3 preview replies
	comments, _, _ := commentRepo.ListByTarget(ctx, "note", note.ID, model.PaginationParams{Page: 1, PageSize: 20})
	assert.Equal(t, 1, len(comments)) // only 1 top-level
	assert.Equal(t, 4, comments[0].ReplyCount)
	assert.Len(t, comments[0].Replies, 3) // preview limit

	// ListReplies should return all 4
	replies, total, _ := commentRepo.ListReplies(ctx, parent.ID, model.PaginationParams{Page: 1, PageSize: 20})
	assert.Equal(t, 4, total)
	assert.Len(t, replies, 4)
}

func TestCommentRepo_Delete(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "delcomment@test.com", Password: "x", Nickname: "Deleter",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Deletable"})

	c, _ := commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "To delete"})
	err := commentRepo.Delete(ctx, c.ID)
	require.NoError(t, err)

	_, err = commentRepo.GetByID(ctx, c.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCommentRepo_Count(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	noteRepo := NewNoteRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "counter@test.com", Password: "x", Nickname: "Counter",
	}, "$2a$10$dummyhash")

	note, _ := noteRepo.Create(ctx, user.ID, model.CreateNoteParams{Title: "Countable"})
	_, _ = commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "One"})
	_, _ = commentRepo.Create(ctx, user.ID, "note", note.ID, model.CreateCommentParams{Content: "Two"})

	count, err := commentRepo.CountByTarget(ctx, "note", note.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
