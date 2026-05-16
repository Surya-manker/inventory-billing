package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/yourusername/inventory-billing/config"
	"go.uber.org/zap"
)

// Server wraps asynq.Server and registers all task handlers in one place.
// Call Server.Run(ctx) in cmd/worker/main.go — it blocks until ctx is cancelled.
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	log    *zap.Logger
}

func NewServer(cfg *config.Config, log *zap.Logger) *Server {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		},
		asynq.Config{
			// Total worker goroutines across all queues.
			Concurrency: cfg.Worker.Concurrency,

			// Priority weights: critical = 60 %, default = 30 %, low = 10 %.
			Queues: map[string]int{
				QueueCritical: 6,
				QueueDefault:  3,
				QueueLow:      1,
			},

			// Exponential back-off: delay(n) = n² seconds (Asynq default).
			RetryDelayFunc: asynq.DefaultRetryDelayFunc,

			// Only count a task as "failed" for real errors, not SkipRetry.
			IsFailure: func(err error) bool {
				return !containsSkipRetry(err)
			},

			// Log every permanent failure at ERROR level.
			ErrorHandler: asynq.ErrorHandlerFunc(
				func(ctx context.Context, task *asynq.Task, err error) {
					log.Error("task permanently failed",
						zap.String("type", task.Type()),
						zap.Error(err),
					)
				},
			),

			Logger: newAsynqLogger(log),
		},
	)

	return &Server{server: srv, mux: asynq.NewServeMux(), log: log}
}

// Register attaches a handler to a task type. Call once per task type
// before Run. The handler is wrapped with the logging middleware automatically.
func (s *Server) Register(taskType string, h asynq.Handler) {
	s.mux.Handle(taskType, withLogging(s.log, h))
}

// Run starts processing tasks and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	if err := s.server.Start(s.mux); err != nil {
		return fmt.Errorf("asynq server start: %w", err)
	}
	<-ctx.Done()
	s.server.Shutdown()
	s.log.Info("asynq server stopped cleanly")
	return nil
}

// ── logging middleware ────────────────────────────────────────────────────────

// withLogging wraps a handler so every task execution is logged with its
// type, outcome, and elapsed duration.
func withLogging(log *zap.Logger, next asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		start := time.Now()
		log.Info("task started", zap.String("type", t.Type()))

		err := next.ProcessTask(ctx, t)
		elapsed := time.Since(start)

		if err != nil {
			log.Warn("task failed",
				zap.String("type", t.Type()),
				zap.Duration("elapsed", elapsed),
				zap.Error(err),
			)
		} else {
			log.Info("task completed",
				zap.String("type", t.Type()),
				zap.Duration("elapsed", elapsed),
			)
		}
		return err
	})
}

// ── zap → asynq.Logger adapter ───────────────────────────────────────────────

type asynqZapLogger struct{ log *zap.Logger }

func newAsynqLogger(log *zap.Logger) asynq.Logger { return &asynqZapLogger{log: log} }

func (l *asynqZapLogger) Debug(args ...any) { l.log.Sugar().Debug(args...) }
func (l *asynqZapLogger) Info(args ...any)  { l.log.Sugar().Info(args...) }
func (l *asynqZapLogger) Warn(args ...any)  { l.log.Sugar().Warn(args...) }
func (l *asynqZapLogger) Error(args ...any) { l.log.Sugar().Error(args...) }
func (l *asynqZapLogger) Fatal(args ...any) { l.log.Sugar().Fatal(args...) }

// ── helpers ───────────────────────────────────────────────────────────────────

// containsSkipRetry walks the error chain looking for asynq.SkipRetry.
func containsSkipRetry(err error) bool {
	for err != nil {
		if err == asynq.SkipRetry {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
