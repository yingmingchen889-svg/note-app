package repo

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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

	_, _ = pool.Exec(context.Background(), "DELETE FROM likes")
	_, _ = pool.Exec(context.Background(), "DELETE FROM comments")
	_, _ = pool.Exec(context.Background(), "DELETE FROM check_ins")
	_, _ = pool.Exec(context.Background(), "DELETE FROM plan_members")
	_, _ = pool.Exec(context.Background(), "DELETE FROM plans")
	_, _ = pool.Exec(context.Background(), "DELETE FROM notes")
	_, _ = pool.Exec(context.Background(), "DELETE FROM users")

	return pool
}
