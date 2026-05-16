package domain

import "time"

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusDead       JobStatus = "dead" // max retries exhausted
)

// JobRecord is an append-only table that tracks every background task the system
// enqueues. It is the DB-side complement to Asynq's Redis-side state and exists
// so that jobs can be queried by the entities they belong to (e.g. "PDF jobs for
// invoice X") and retained long after Asynq's own retention window expires.
type JobRecord struct {
	ID          string    `gorm:"type:varchar(36);primaryKey"             json:"id"` // Asynq task ID
	Type        string    `gorm:"type:varchar(100);not null;index"         json:"type"`
	Status      JobStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	Queue       string    `gorm:"type:varchar(50);not null;default:'default'" json:"queue"`
	EntityType  string    `gorm:"type:varchar(50);not null;index:idx_job_entity" json:"entity_type"`
	EntityID    string    `gorm:"type:varchar(100);not null;index:idx_job_entity" json:"entity_id"`
	Payload     string    `gorm:"type:json;not null"                       json:"payload"`
	Result      string    `gorm:"type:json"                                json:"result,omitempty"`
	ErrorMsg    string    `gorm:"type:text"                                json:"error_msg,omitempty"`
	RetryCount  int       `gorm:"not null;default:0"                       json:"retry_count"`
	MaxRetries  int       `gorm:"not null;default:3"                       json:"max_retries"`
	EnqueuedAt  time.Time `gorm:"not null"                                 json:"enqueued_at"`
	StartedAt   *time.Time `                                                json:"started_at,omitempty"`
	CompletedAt *time.Time `                                                json:"completed_at,omitempty"`
	CreatedBy   string    `gorm:"type:varchar(36)"                         json:"created_by,omitempty"`
	CreatedAt   time.Time `                                                json:"created_at"`
	UpdatedAt   time.Time `                                                json:"updated_at"`
}
