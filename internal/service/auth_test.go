package service

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://noteapp:noteapp@localhost:5432/noteapp?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	_, _ = pool.Exec(context.Background(), "DELETE FROM check_ins")
	_, _ = pool.Exec(context.Background(), "DELETE FROM plan_members")
	_, _ = pool.Exec(context.Background(), "DELETE FROM plans")
	_, _ = pool.Exec(context.Background(), "DELETE FROM notes")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users")
	return pool
}

func TestAuthService_RegisterAndLogin(t *testing.T) {
	pool := testPool(t)
	userRepo := repo.NewUserRepo(pool)
	svc := NewAuthService(userRepo, "test-secret", 72)
	ctx := context.Background()

	user, token, err := svc.Register(ctx, model.RegisterParams{
		Email:    "auth@test.com",
		Password: "password123",
		Nickname: "Auth Tester",
	})
	require.NoError(t, err)
	assert.Equal(t, "auth@test.com", user.Email)
	assert.NotEmpty(t, token)

	user2, token2, err := svc.Login(ctx, model.LoginParams{
		Email:    "auth@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, user.ID, user2.ID)
	assert.NotEmpty(t, token2)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	pool := testPool(t)
	userRepo := repo.NewUserRepo(pool)
	svc := NewAuthService(userRepo, "test-secret", 72)
	ctx := context.Background()

	_, _, _ = svc.Register(ctx, model.RegisterParams{
		Email: "wrong@test.com", Password: "correct", Nickname: "test",
	})

	_, _, err := svc.Login(ctx, model.LoginParams{
		Email: "wrong@test.com", Password: "incorrect",
	})
	assert.Error(t, err)
}
