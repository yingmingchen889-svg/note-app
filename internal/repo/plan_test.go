package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/note-app/internal/model"
)

func TestPlanRepo_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "plantest@test.com", Password: "x", Nickname: "Planner",
	}, "$2a$10$dummyhash")

	plan, err := planRepo.Create(ctx, user.ID, model.CreatePlanParams{
		Title:     "Daily Exercise",
		StartDate: "2026-03-19",
	})
	require.NoError(t, err)
	assert.Equal(t, "Daily Exercise", plan.Title)
	assert.Equal(t, "private", plan.Visibility)

	found, err := planRepo.GetByID(ctx, plan.ID)
	require.NoError(t, err)
	assert.Equal(t, plan.ID, found.ID)
}

func TestPlanRepo_CreateAddsOwnerMember(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "owntest@test.com", Password: "x", Nickname: "Owner",
	}, "$2a$10$dummyhash")

	plan, _ := planRepo.Create(ctx, user.ID, model.CreatePlanParams{
		Title: "Test Plan", StartDate: "2026-03-19",
	})

	members, err := planRepo.ListMembers(ctx, plan.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, "owner", members[0].Role)
}

func TestPlanRepo_ListByUser(t *testing.T) {
	pool := testPool(t)
	userRepo := NewUserRepo(pool)
	planRepo := NewPlanRepo(pool)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, model.RegisterParams{
		Email: "listplan@test.com", Password: "x", Nickname: "Lister",
	}, "$2a$10$dummyhash")

	_, _ = planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Plan A", StartDate: "2026-03-19"})
	_, _ = planRepo.Create(ctx, user.ID, model.CreatePlanParams{Title: "Plan B", StartDate: "2026-03-19"})

	plans, total, err := planRepo.ListByUser(ctx, user.ID, model.PaginationParams{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, plans, 2)
}
