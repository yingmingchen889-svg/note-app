package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type PlanService struct {
	planRepo *repo.PlanRepo
}

func NewPlanService(planRepo *repo.PlanRepo) *PlanService {
	return &PlanService{planRepo: planRepo}
}

func (s *PlanService) Create(ctx context.Context, userID uuid.UUID, params model.CreatePlanParams) (*model.Plan, error) {
	return s.planRepo.Create(ctx, userID, params)
}

func (s *PlanService) GetByID(ctx context.Context, userID uuid.UUID, planID uuid.UUID) (*model.Plan, error) {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.Visibility == "private" && plan.UserID != userID {
		return nil, ErrForbidden
	}
	return plan, nil
}

func (s *PlanService) List(ctx context.Context, userID uuid.UUID, params model.PaginationParams) ([]model.Plan, int, error) {
	return s.planRepo.ListByUser(ctx, userID, params)
}

func (s *PlanService) Update(ctx context.Context, userID uuid.UUID, planID uuid.UUID, params model.UpdatePlanParams) (*model.Plan, error) {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.UserID != userID {
		return nil, ErrForbidden
	}
	return s.planRepo.Update(ctx, planID, params)
}

func (s *PlanService) Share(ctx context.Context, userID uuid.UUID, planID uuid.UUID) (*model.Plan, error) {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan.UserID != userID {
		return nil, ErrForbidden
	}
	newVis := "public"
	if plan.Visibility == "public" {
		newVis = "private"
	}
	return s.planRepo.UpdateVisibility(ctx, planID, newVis)
}

func (s *PlanService) Join(ctx context.Context, userID uuid.UUID, planID uuid.UUID) error {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return err
	}
	if plan.Visibility != "public" {
		return ErrForbidden
	}
	return s.planRepo.AddMember(ctx, planID, userID)
}

func (s *PlanService) ListMembers(ctx context.Context, planID uuid.UUID) ([]model.PlanMember, error) {
	return s.planRepo.ListMembers(ctx, planID)
}
