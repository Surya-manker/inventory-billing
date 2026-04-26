package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"github.com/yourusername/inventory-billing/pkg/mailer"
	"github.com/yourusername/inventory-billing/pkg/storage"
	"go.uber.org/zap"
)

// EmailHandler is the Asynq worker handler for TypeEmailSend tasks.
// It renders the HTML body, optionally fetches an attachment from storage,
// and sends the email via the injected Mailer.
type EmailHandler struct {
	mailer  mailer.Mailer
	stor    storage.Storage
	jobRepo repository.JobRepository
	log     *zap.Logger
}

func NewEmailHandler(
	m mailer.Mailer,
	stor storage.Storage,
	jobRepo repository.JobRepository,
	log *zap.Logger,
) *EmailHandler {
	return &EmailHandler{mailer: m, stor: stor, jobRepo: jobRepo, log: log}
}

// ProcessTask implements asynq.Handler.
func (h *EmailHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p EmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("%w: unmarshal email payload: %v", asynq.SkipRetry, err)
	}

	taskID := t.ResultWriter().TaskID()
	log := h.log.With(
		zap.String("task_id", taskID),
		zap.String("to", p.To),
		zap.String("template", p.TemplateName),
	)

	// ── 1. Mark processing ───────────────────────────────────────────────────
	now := time.Now()
	_ = h.jobRepo.Update(ctx, taskID, repository.UpdateJobParams{
		Status:    domain.JobStatusProcessing,
		StartedAt: &now,
	})

	// ── 2. Render HTML body ──────────────────────────────────────────────────
	htmlBody, err := h.renderBody(p)
	if err != nil {
		// Template render errors are permanent; retrying won't help.
		return h.failPermanent(ctx, taskID, log, fmt.Sprintf("render template: %v", err))
	}

	// ── 3. Build mailer.Message ──────────────────────────────────────────────
	msg := mailer.Message{
		To:      p.To,
		Subject: p.Subject,
		HTMLBody: htmlBody,
	}

	if p.AttachStorKey != "" {
		pdfBytes, err := h.stor.Get(ctx, p.AttachStorKey)
		if err != nil {
			log.Warn("attachment not found in storage, sending without attachment",
				zap.String("key", p.AttachStorKey),
				zap.Error(err),
			)
			// Don't fail the whole email if the PDF attachment is missing.
		} else {
			msg.Attachments = append(msg.Attachments, mailer.Attachment{
				Filename:    p.AttachName,
				Data:        pdfBytes,
				ContentType: "application/pdf",
			})
		}
	}

	// ── 4. Send ──────────────────────────────────────────────────────────────
	if err := h.mailer.Send(ctx, msg); err != nil {
		log.Warn("email send failed, will retry", zap.Error(err))
		_ = h.jobRepo.Update(ctx, taskID, repository.UpdateJobParams{
			Status:    domain.JobStatusFailed,
			ErrorMsg:  err.Error(),
			IncrRetry: true,
		})
		return err // retryable
	}

	// ── 5. Mark completed ────────────────────────────────────────────────────
	completedAt := time.Now()
	_ = h.jobRepo.Update(ctx, taskID, repository.UpdateJobParams{
		Status:      domain.JobStatusCompleted,
		CompletedAt: &completedAt,
	})
	log.Info("email sent successfully")
	return nil
}

func (h *EmailHandler) renderBody(p EmailPayload) (string, error) {
	switch p.TemplateName {
	case "invoice_created":
		data := mailer.InvoiceEmailData{
			CustomerName:  strVal(p.TemplateData, "CustomerName"),
			InvoiceNumber: strVal(p.TemplateData, "InvoiceNumber"),
			TotalAmount:   strVal(p.TemplateData, "TotalAmount"),
			DueDate:       strVal(p.TemplateData, "DueDate"),
			CompanyName:   strVal(p.TemplateData, "CompanyName"),
		}
		return mailer.BuildInvoiceEmail(data)
	default:
		// Generic fallback: render template_data as a simple HTML table.
		return h.renderGenericBody(p), nil
	}
}

func (h *EmailHandler) renderGenericBody(p EmailPayload) string {
	body := "<html><body>"
	body += "<h2>" + p.Subject + "</h2><table>"
	for k, v := range p.TemplateData {
		body += fmt.Sprintf("<tr><td><b>%s</b></td><td>%v</td></tr>", k, v)
	}
	body += "</table></body></html>"
	return body
}

func (h *EmailHandler) failPermanent(ctx context.Context, taskID string, log *zap.Logger, msg string) error {
	log.Error("email job failed permanently", zap.String("reason", msg))
	_ = h.jobRepo.Update(ctx, taskID, repository.UpdateJobParams{
		Status:   domain.JobStatusDead,
		ErrorMsg: msg,
	})
	return fmt.Errorf("%w: %s", asynq.SkipRetry, msg)
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
