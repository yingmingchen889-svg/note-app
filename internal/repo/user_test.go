package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestUserRepo_CreateAndGetByEmail(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	user, err := repo.Create(ctx, model.RegisterParams{
		Email:    "test@example.com",
		Password: "hashedpassword",
		Nickname: "tester",
	}, "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ01234")
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "tester", user.Nickname)
	assert.NotEmpty(t, user.ID)

	found, err := repo.GetByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
	assert.ErrorIs(t, err, ErrNotFound)
}
