package repository

import (
	"context"
	"time"

	"github.com/yourusername/inventory-billing/internal/domain"
	"gorm.io/gorm"
)

type UpdateJobParams struct {
	Status      domain.JobStatus
	StartedAt   *time.Time
	CompletedAt *time.Time
	ErrorMsg    string
	Result      string
	IncrRetry   bool
}

type JobRepository interface {
	Create(ctx context.Context, job *domain.JobRecord) error
	FindByID(ctx context.Context, id string) (*domain.JobRecord, error)
	FindByEntity(ctx context.Context, entityType, entityID string) ([]domain.JobRecord, error)
	Update(ctx context.Context, id string, p UpdateJobParams) error
}

type jobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) JobRepository {
	return &jobRepository{db: db}
}

func (r *jobRepository) Create(ctx context.Context, job *domain.JobRecord) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *jobRepository) FindByID(ctx context.Context, id string) (*domain.JobRecord, error) {
	var job domain.JobRecord
	if err := r.db.WithContext(ctx).First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *jobRepository) FindByEntity(ctx context.Context, entityType, entityID string) ([]domain.JobRecord, error) {
	var jobs []domain.JobRecord
	err := r.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("enqueued_at DESC").
		Find(&jobs).Error
	return jobs, err
}

func (r *jobRepository) Update(ctx context.Context, id string, p UpdateJobParams) error {
	updates := map[string]any{"status": p.Status}
	if p.StartedAt != nil {
		updates["started_at"] = p.StartedAt
	}
	if p.CompletedAt != nil {
		updates["completed_at"] = p.CompletedAt
	}
	if p.ErrorMsg != "" {
		updates["error_msg"] = p.ErrorMsg
	}
	if p.Result != "" {
		updates["result"] = p.Result
	}
	if p.IncrRetry {
		updates["retry_count"] = gorm.Expr("retry_count + 1")
	}
	return r.db.WithContext(ctx).
		Model(&domain.JobRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}
