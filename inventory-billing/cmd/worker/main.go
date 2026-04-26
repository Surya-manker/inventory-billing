// cmd/worker — background job runner.
// Run alongside the API server:  go run ./cmd/worker
//
// The worker and the API share the same database and Redis connection but are
// separate OS processes so a slow job never blocks an HTTP request.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/yourusername/inventory-billing/config"
	"github.com/yourusername/inventory-billing/pkg/cache"
	"github.com/yourusername/inventory-billing/pkg/database"
	"github.com/yourusername/inventory-billing/pkg/jobs"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	rdb := cache.NewRedisClient(cfg)

	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	runner := jobs.NewRunner(logger,
		jobs.NewLowStockAlertJob(db, logger),     // every hour
		jobs.NewOverdueInvoiceJob(db, logger),    // every 24 hours
		jobs.NewDailySnapshotJob(db, rdb, logger), // every 24 hours
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runner.Start(ctx) // blocks until signal
	logger.Info("worker exited cleanly")
}
