package main

import (
	"log"

	"github.com/yourusername/inventory-billing/config"
	"github.com/yourusername/inventory-billing/internal/router"
	"github.com/yourusername/inventory-billing/pkg/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	r := router.Setup(db, cfg)

	log.Printf("server listening on %s", cfg.Server.Address)
	if err := r.Run(cfg.Server.Address); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
