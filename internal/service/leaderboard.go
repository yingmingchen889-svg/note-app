package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/user/note-app/internal/model"
	"github.com/user/note-app/internal/repo"
)

type LeaderboardService struct {
	rdb         *redis.Client
	checkInRepo *repo.CheckInRepo
	userRepo    *repo.UserRepo
}

func NewLeaderboardService(rdb *redis.Client, checkInRepo *repo.CheckInRepo, userRepo *repo.UserRepo) *LeaderboardService {
	return &LeaderboardService{rdb: rdb, checkInRepo: checkInRepo, userRepo: userRepo}
}

func leaderboardKey(planID uuid.UUID) string {
	return fmt.Sprintf("plan:%s:leaderboard", planID.String())
}

// IncrementScore increments the user's check-in count in the leaderboard.
func (s *LeaderboardService) IncrementScore(ctx context.Context, planID, userID uuid.UUID) error {
	return s.rdb.ZIncrBy(ctx, leaderboardKey(planID), 1, userID.String()).Err()
}

// GetLeaderboard returns top N users by check-in count for a plan.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context, planID uuid.UUID, limit int) ([]model.LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Get top N from Redis
	results, err := s.rdb.ZRevRangeWithScores(ctx, leaderboardKey(planID), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]model.LeaderboardEntry, 0, len(results))
	for i, z := range results {
		userIDStr, _ := z.Member.(string)
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}

		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			continue
		}

		// Calculate streak from PostgreSQL
		today := time.Now().Format("2006-01-02")
		streak, _ := s.checkInRepo.CurrentStreak(ctx, planID, userID, today)

		entries = append(entries, model.LeaderboardEntry{
			Rank: i + 1,
			User: model.UserBrief{
				ID:        user.ID,
				Nickname:  user.Nickname,
				AvatarURL: user.AvatarURL,
			},
			CheckInCount: int(z.Score),
			Streak:       streak,
		})
	}
	return entries, nil
}
