package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type CheckInService struct {
	checkInRepo *repo.CheckInRepo
	planRepo    *repo.PlanRepo
}

func NewCheckInService(checkInRepo *repo.CheckInRepo, planRepo *repo.PlanRepo) *CheckInService {
	return &CheckInService{checkInRepo: checkInRepo, planRepo: planRepo}
}

func (s *CheckInService) CheckIn(ctx context.Context, userID uuid.UUID, planID uuid.UUID, params model.UpsertCheckInParams) (*model.CheckIn, error) {
	// Verify user is a member of the plan
	isMember, err := s.planRepo.IsMember(ctx, planID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrForbidden
	}

	today := time.Now().Format("2006-01-02")
	return s.checkInRepo.Upsert(ctx, planID, userID, today, params)
}

func (s *CheckInService) ListByPlan(ctx context.Context, planID uuid.UUID, params model.PaginationParams) ([]model.CheckIn, int, error) {
	return s.checkInRepo.ListByPlan(ctx, planID, params)
}

func (s *CheckInService) Calendar(ctx context.Context, userID uuid.UUID, startDate, endDate string) ([]model.CalendarEntry, error) {
	return s.checkInRepo.Calendar(ctx, userID, startDate, endDate)
}

func (s *CheckInService) Streak(ctx context.Context, planID, userID uuid.UUID) (int, error) {
	today := time.Now().Format("2006-01-02")
	return s.checkInRepo.CurrentStreak(ctx, planID, userID, today)
}
