package main

import (
	"context"
	"log"

	"github.com/user/note-app/internal/config"
	"github.com/user/note-app/internal/handler"
	"github.com/user/note-app/internal/repo"
	"github.com/user/note-app/internal/service"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := repo.NewPool(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	userRepo := repo.NewUserRepo(pool)
	noteRepo := repo.NewNoteRepo(pool)
	planRepo := repo.NewPlanRepo(pool)
	checkInRepo := repo.NewCheckInRepo(pool)

	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpireHours)
	noteService := service.NewNoteService(noteRepo)
	planService := service.NewPlanService(planRepo)
	checkInService := service.NewCheckInService(checkInRepo, planRepo)

	handlers := &handler.Handlers{
		Auth:      handler.NewAuthHandler(authService),
		Note:      handler.NewNoteHandler(noteService),
		Plan:      handler.NewPlanHandler(planService),
		CheckIn:   handler.NewCheckInHandler(checkInService),
		JWTSecret: cfg.JWTSecret,
	}
	r := handler.SetupRouter(handlers)

	log.Printf("Starting server on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
