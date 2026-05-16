// cmd/worker — background job processor.
//
// This process runs two concurrent systems:
//  1. Asynq server  — processes tasks enqueued by the API (PDF generation, email).
//  2. Scheduler     — fires time-based recurring jobs (low-stock alerts, snapshots).
//
// Run alongside the API server:  go run ./cmd/worker
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/yourusername/inventory-billing/config"
	"github.com/yourusername/inventory-billing/internal/repository"
	"github.com/yourusername/inventory-billing/pkg/cache"
	"github.com/yourusername/inventory-billing/pkg/database"
	"github.com/yourusername/inventory-billing/pkg/jobs"
	"github.com/yourusername/inventory-billing/pkg/mailer"
	"github.com/yourusername/inventory-billing/pkg/pdf"
	"github.com/yourusername/inventory-billing/pkg/queue"
	"github.com/yourusername/inventory-billing/pkg/storage"
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

	// ── storage backend ───────────────────────────────────────────────────────
	var stor storage.Storage
	switch cfg.Storage.Provider {
	case "local":
		stor, err = storage.NewLocalStorage(cfg.Storage.LocalDir, cfg.Storage.LocalURL)
		if err != nil {
			log.Fatalf("storage: %v", err)
		}
	default:
		log.Fatalf("unknown storage provider: %s", cfg.Storage.Provider)
	}

	// ── mailer ────────────────────────────────────────────────────────────────
	var m mailer.Mailer
	switch cfg.Mail.Provider {
	case "smtp":
		m = mailer.NewSMTPMailer(mailer.SMTPConfig{
			Host:     cfg.Mail.SMTPHost,
			Port:     cfg.Mail.SMTPPort,
			Username: cfg.Mail.SMTPUser,
			Password: cfg.Mail.SMTPPass,
			From:     cfg.Mail.From,
			UseTLS:   cfg.Mail.UseTLS,
		})
	default:
		logger.Warn("mail provider not configured, using noop (emails will be discarded)",
			zap.String("provider", cfg.Mail.Provider))
		m = mailer.NoopMailer{}
	}

	// ── repositories ─────────────────────────────────────────────────────────
	jobRepo     := repository.NewJobRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)

	// ── queue client (for PDF handler → email enqueue) ────────────────────────
	queueClient := queue.NewClient(cfg, jobRepo, logger)
	defer queueClient.Close() //nolint:errcheck

	// ── task handlers ─────────────────────────────────────────────────────────
	pdfGen      := pdf.NewGenerator()
	pdfHandler  := queue.NewInvoicePDFHandler(invoiceRepo, jobRepo, pdfGen, stor, queueClient, cfg, logger)
	emailHandler := queue.NewEmailHandler(m, stor, jobRepo, logger)

	// ── Asynq server ──────────────────────────────────────────────────────────
	asynqServer := queue.NewServer(cfg, logger)
	asynqServer.Register(queue.TypeInvoicePDF, pdfHandler)
	asynqServer.Register(queue.TypeEmailSend, emailHandler)

	// ── scheduler for time-based recurring jobs ───────────────────────────────
	scheduler := jobs.NewRunner(logger,
		jobs.NewLowStockAlertJob(db, logger),
		jobs.NewOverdueInvoiceJob(db, logger),
		jobs.NewDailySnapshotJob(db, rdb, logger),
	)

	// ── graceful shutdown ─────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run the scheduler in its own goroutine.
	go scheduler.Start(ctx)

	// Run blocks until ctx is cancelled.
	if err := asynqServer.Run(ctx); err != nil {
		logger.Error("asynq server error", zap.Error(err))
	}
	logger.Info("worker exited cleanly")
}
