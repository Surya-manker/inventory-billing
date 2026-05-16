// Package jobs provides a simple in-process job scheduler.
//
// Architecture note:
// For a single-instance deployment this runner is sufficient.
// When you scale to multiple instances, replace it with a Redis-backed
// distributed queue such as github.com/hibiken/asynq.  The Job interface
// is stable — only the Runner implementation changes.
package jobs

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Job is the interface every background task must implement.
type Job interface {
	// Name returns a human-readable label used in logs and metrics.
	Name() string
	// Schedule returns how often the job runs.
	Schedule() time.Duration
	// Execute runs the job. A non-nil error is logged; it never stops the runner.
	Execute(ctx context.Context) error
}

// Runner starts each job in its own goroutine and respects context cancellation.
type Runner struct {
	jobs []Job
	log  *zap.Logger
}

func NewRunner(log *zap.Logger, jobs ...Job) *Runner {
	return &Runner{jobs: jobs, log: log}
}

// Start launches all jobs concurrently and blocks until ctx is cancelled.
func (r *Runner) Start(ctx context.Context) {
	for _, job := range r.jobs {
		go r.loop(ctx, job)
	}
	r.log.Info("job runner started", zap.Int("jobs", len(r.jobs)))
	<-ctx.Done()
	r.log.Info("job runner stopped")
}

func (r *Runner) loop(ctx context.Context, job Job) {
	// Execute once immediately so we don't wait a full interval on startup.
	r.execute(ctx, job)

	ticker := time.NewTicker(job.Schedule())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.execute(ctx, job)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Runner) execute(ctx context.Context, job Job) {
	start := time.Now()
	if err := job.Execute(ctx); err != nil {
		r.log.Error("job failed",
			zap.String("job", job.Name()),
			zap.Duration("elapsed", time.Since(start)),
			zap.Error(err),
		)
		return
	}
	r.log.Info("job completed",
		zap.String("job", job.Name()),
		zap.Duration("elapsed", time.Since(start)),
	)
}
