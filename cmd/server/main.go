package main

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/user/note-app/internal/config"
	"github.com/user/note-app/internal/handler"
	"github.com/user/note-app/internal/repo"
	"github.com/user/note-app/internal/service"
	"github.com/user/note-app/internal/storage"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	minioClient, err := storage.NewMinIOClient(cfg.MinIO)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	pool, err := repo.NewPool(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Repos
	userRepo := repo.NewUserRepo(pool)
	noteRepo := repo.NewNoteRepo(pool)
	planRepo := repo.NewPlanRepo(pool)
	checkInRepo := repo.NewCheckInRepo(pool)
	likeRepo := repo.NewLikeRepo(pool)
	commentRepo := repo.NewCommentRepo(pool)
	exploreRepo := repo.NewExploreRepo(pool)
	growthRepo := repo.NewGrowthRepo(pool)

	// Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpireHours)
	noteService := service.NewNoteService(noteRepo)
	planService := service.NewPlanService(planRepo)
	leaderboardService := service.NewLeaderboardService(rdb, checkInRepo, userRepo)
	checkInService := service.NewCheckInService(checkInRepo, planRepo, leaderboardService)
	socialService := service.NewSocialService(likeRepo, commentRepo, noteRepo, planRepo, checkInRepo)
	growthService := service.NewGrowthService(pool, growthRepo)

	// Handlers + Router
	handlers := &handler.Handlers{
		Auth:      handler.NewAuthHandler(authService),
		Note:      handler.NewNoteHandler(noteService, socialService),
		Plan:      handler.NewPlanHandler(planService, leaderboardService, socialService),
		CheckIn:   handler.NewCheckInHandler(checkInService),
		Upload:    handler.NewUploadHandler(minioClient),
		Social:    handler.NewSocialHandler(socialService),
		Explore:   handler.NewExploreHandler(exploreRepo),
		Growth:    handler.NewGrowthHandler(growthService),
		JWTSecret: cfg.JWTSecret,
	}
	r := handler.SetupRouter(handlers)

	log.Printf("Starting server on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
