package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestCheckInRepo_Upsert(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	checkInRepo := NewCheckInRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "checkin@test.com", Password: "x", Nickname: "Checker",
	}, "$2a$10$dummyhash")

	plan, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{
		Title: "Test Plan", StartDate: "2026-03-19",
	})

	// First check-in
	ci, _, err := checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-19", model.UpsertCheckInParams{
		Content: "First check-in",
	})
	require.NoError(t, err)
	assert.Equal(t, "First check-in", ci.Content)

	// Upsert same date — should overwrite
	ci2, _, err := checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-19", model.UpsertCheckInParams{
		Content: "Updated check-in",
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated check-in", ci2.Content)
	assert.Equal(t, ci.ID, ci2.ID) // same record updated
}

func TestCheckInRepo_ListByPlan(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	checkInRepo := NewCheckInRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "listci@test.com", Password: "x", Nickname: "Lister",
	}, "$2a$10$dummyhash")

	plan, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{
		Title: "List Plan", StartDate: "2026-03-01",
	})

	_, _, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-01", model.UpsertCheckInParams{Content: "Day 1"})
	_, _, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-02", model.UpsertCheckInParams{Content: "Day 2"})

	checkins, total, err := checkInRepo.ListByPlan(ctx, plan.ID, model.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, checkins, 2)
}

func TestCheckInRepo_Calendar(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	checkInRepo := NewCheckInRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "cal@test.com", Password: "x", Nickname: "Calendar",
	}, "$2a$10$dummyhash")

	plan1, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Plan A", StartDate: "2026-03-01"})
	plan2, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Plan B", StartDate: "2026-03-01"})

	_, _, _ = checkInRepo.Upsert(ctx, plan1.ID, user.ID, "2026-03-01", model.UpsertCheckInParams{Content: "A1"})
	_, _, _ = checkInRepo.Upsert(ctx, plan2.ID, user.ID, "2026-03-01", model.UpsertCheckInParams{Content: "B1"})
	_, _, _ = checkInRepo.Upsert(ctx, plan1.ID, user.ID, "2026-03-02", model.UpsertCheckInParams{Content: "A2"})

	entries, err := checkInRepo.Calendar(ctx, user.ID, "2026-03-01", "2026-03-31")
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestCheckInRepo_Streak(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	checkInRepo := NewCheckInRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "streak@test.com", Password: "x", Nickname: "Streak",
	}, "$2a$10$dummyhash")

	plan, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Streak Plan", StartDate: "2026-03-01"})

	_, _, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-17", model.UpsertCheckInParams{Content: "d1"})
	_, _, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-18", model.UpsertCheckInParams{Content: "d2"})
	_, _, _ = checkInRepo.Upsert(ctx, plan.ID, user.ID, "2026-03-19", model.UpsertCheckInParams{Content: "d3"})

	streak, err := checkInRepo.CurrentStreak(ctx, plan.ID, user.ID, "2026-03-19")
	require.NoError(t, err)
	assert.Equal(t, 3, streak)
}
