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
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpireHours)

	handlers := &handler.Handlers{
		Auth:      handler.NewAuthHandler(authService),
		JWTSecret: cfg.JWTSecret,
	}
	r := handler.SetupRouter(handlers)

	log.Printf("Starting server on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
