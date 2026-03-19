package main

import (
	"log"

	"github.com/user/note-app/internal/config"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting server on port %s", cfg.ServerPort)
}
