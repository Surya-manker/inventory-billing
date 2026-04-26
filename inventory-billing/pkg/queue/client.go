package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/yourusername/inventory-billing/config"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"go.uber.org/zap"
)

// Client wraps asynq.Client and adds typed enqueue methods. Each method:
//  1. Serialises the typed payload
//  2. Enqueues the Asynq task (writes to Redis)
//  3. Persists a JobRecord row in MySQL so the job is queryable by entity ID
//     and survives Redis flushes / Asynq retention expiry.
type Client struct {
	asynq   *asynq.Client
	jobRepo repository.JobRepository
	log     *zap.Logger
}

func NewClient(cfg *config.Config, jobRepo repository.JobRepository, log *zap.Logger) *Client {
	return &Client{
		asynq: asynq.NewClient(asynq.RedisClientOpt{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}),
		jobRepo: jobRepo,
		log:     log,
	}
}

// Close releases the underlying Redis connection. Call in defer from main.
func (c *Client) Close() error { return c.asynq.Close() }

// ── typed enqueue methods ─────────────────────────────────────────────────────

// EnqueueInvoicePDF enqueues a PDF-generation job for the given invoice.
// Returns the Asynq task ID which the caller can expose to the client for polling.
func (c *Client) EnqueueInvoicePDF(ctx context.Context, p InvoicePDFPayload) (taskID string, err error) {
	return c.enqueue(ctx, enqueueParams{
		taskType:    TypeInvoicePDF,
		queue:       QueueCritical,
		payload:     p,
		entityType:  "invoice",
		entityID:    p.InvoiceID,
		createdBy:   p.RequestedBy,
		maxRetries:  3,
		timeout:     2 * time.Minute,
		retention:   24 * time.Hour,
	})
}

// EnqueueEmail enqueues a transactional email job.
func (c *Client) EnqueueEmail(ctx context.Context, p EmailPayload) (taskID string, err error) {
	entityID := p.InvoiceID
	if entityID == "" {
		entityID = "system"
	}
	return c.enqueue(ctx, enqueueParams{
		taskType:   TypeEmailSend,
		queue:      QueueDefault,
		payload:    p,
		entityType: "email",
		entityID:   entityID,
		createdBy:  p.RequestedBy,
		maxRetries: 5,
		timeout:    30 * time.Second,
		retention:  24 * time.Hour,
	})
}

// ── internal helpers ──────────────────────────────────────────────────────────

type enqueueParams struct {
	taskType   string
	queue      string
	payload    any
	entityType string
	entityID   string
	createdBy  string
	maxRetries int
	timeout    time.Duration
	retention  time.Duration
	// processAt: zero means "immediately"
	processAt time.Time
}

func (c *Client) enqueue(ctx context.Context, p enqueueParams) (string, error) {
	raw, err := json.Marshal(p.payload)
	if err != nil {
		return "", fmt.Errorf("queue: marshal payload for %s: %w", p.taskType, err)
	}

	opts := []asynq.Option{
		asynq.Queue(p.queue),
		asynq.MaxRetry(p.maxRetries),
		asynq.Timeout(p.timeout),
		asynq.Retention(p.retention),
	}
	if !p.processAt.IsZero() {
		opts = append(opts, asynq.ProcessAt(p.processAt))
	}

	task := asynq.NewTask(p.taskType, raw, opts...)

	info, err := c.asynq.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("queue: enqueue %s: %w", p.taskType, err)
	}

	// Best-effort DB record — a failure here must NOT block the caller.
	job := &domain.JobRecord{
		ID:         info.ID,
		Type:       p.taskType,
		Status:     domain.JobStatusPending,
		Queue:      info.Queue,
		EntityType: p.entityType,
		EntityID:   p.entityID,
		Payload:    string(raw),
		MaxRetries: p.maxRetries,
		EnqueuedAt: time.Now(),
		CreatedBy:  p.createdBy,
	}
	if dbErr := c.jobRepo.Create(ctx, job); dbErr != nil {
		c.log.Warn("enqueue: failed to persist job record",
			zap.String("task_id", info.ID),
			zap.String("type", p.taskType),
			zap.Error(dbErr),
		)
	}

	c.log.Info("job enqueued",
		zap.String("task_id", info.ID),
		zap.String("type", p.taskType),
		zap.String("queue", info.Queue),
		zap.String("entity_id", p.entityID),
	)
	return info.ID, nil
}
