package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/yourusername/inventory-billing/config"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/internal/repository"
	"github.com/yourusername/inventory-billing/pkg/pdf"
	"github.com/yourusername/inventory-billing/pkg/storage"
	"go.uber.org/zap"
)

// InvoicePDFHandler is the Asynq worker handler for TypeInvoicePDF tasks.
// It generates a PDF for the given invoice and stores it using the Storage
// backend, then enqueues a follow-up email job.
type InvoicePDFHandler struct {
	invoiceRepo repository.InvoiceRepository
	jobRepo     repository.JobRepository
	pdfGen      pdf.Generator
	stor        storage.Storage
	client      *Client // enqueues the follow-up email job
	cfg         *config.Config
	log         *zap.Logger
}

func NewInvoicePDFHandler(
	invoiceRepo repository.InvoiceRepository,
	jobRepo repository.JobRepository,
	pdfGen pdf.Generator,
	stor storage.Storage,
	client *Client,
	cfg *config.Config,
	log *zap.Logger,
) *InvoicePDFHandler {
	return &InvoicePDFHandler{
		invoiceRepo: invoiceRepo,
		jobRepo:     jobRepo,
		pdfGen:      pdfGen,
		stor:        stor,
		client:      client,
		cfg:         cfg,
		log:         log,
	}
}

// ProcessTask implements asynq.Handler.
// Asynq calls this method in a worker goroutine. Any returned error triggers a
// retry according to the task's MaxRetry setting. Wrap unrecoverable errors
// with asynq.SkipRetry to move the task directly to the dead queue.
func (h *InvoicePDFHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p InvoicePDFPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		// Malformed payload will never be fixed by retrying.
		return fmt.Errorf("%w: unmarshal payload: %v", asynq.SkipRetry, err)
	}

	taskID := t.ResultWriter().TaskID()
	log := h.log.With(
		zap.String("task_id", taskID),
		zap.String("invoice_id", p.InvoiceID),
	)

	// ── 1. Mark job as processing ────────────────────────────────────────────
	now := time.Now()
	_ = h.jobRepo.Update(ctx, taskID, repository.UpdateJobParams{
		Status:    domain.JobStatusProcessing,
		StartedAt: &now,
	})

	// ── 2. Load the invoice ──────────────────────────────────────────────────
	invoiceID, err := uuid.Parse(p.InvoiceID)
	if err != nil {
		return h.failPermanent(ctx, taskID, log, "invalid invoice_id: "+p.InvoiceID)
	}

	invoice, err := h.invoiceRepo.FindByID(ctx, invoiceID)
	if err != nil {
		if isNotFound(err) {
			return h.failPermanent(ctx, taskID, log, "invoice not found: "+p.InvoiceID)
		}
		return h.failRetryable(ctx, taskID, log, fmt.Errorf("load invoice: %w", err))
	}

	// ── 3. Build PDF data ────────────────────────────────────────────────────
	data := h.buildPDFData(invoice)

	// ── 4. Generate the PDF ──────────────────────────────────────────────────
	pdfBytes, err := h.pdfGen.Generate(data)
	if err != nil {
		return h.failRetryable(ctx, taskID, log, fmt.Errorf("generate pdf: %w", err))
	}
	log.Info("pdf generated", zap.Int("bytes", len(pdfBytes)))

	// ── 5. Store the PDF ─────────────────────────────────────────────────────
	storKey := fmt.Sprintf("invoices/%s/%s.pdf", invoice.InvoiceNumber, invoice.ID.String())
	if err := h.stor.Put(ctx, storKey, "application/pdf", pdfBytes); err != nil {
		return h.failRetryable(ctx, taskID, log, fmt.Errorf("store pdf: %w", err))
	}

	pdfURL := h.stor.PublicURL(storKey)
	log.Info("pdf stored", zap.String("url", pdfURL))

	// ── 6. Mark job completed ────────────────────────────────────────────────
	result, _ := json.Marshal(InvoicePDFResult{PDFURL: pdfURL, StorKey: storKey})
	completedAt := time.Now()
	_ = h.jobRepo.Update(ctx, taskID, repository.UpdateJobParams{
		Status:      domain.JobStatusCompleted,
		CompletedAt: &completedAt,
		Result:      string(result),
	})

	// ── 7. Enqueue follow-up email (non-blocking, best-effort) ───────────────
	h.enqueueInvoiceEmail(ctx, invoice, pdfURL, storKey, p.RequestedBy, log)

	return nil
}

// buildPDFData maps a domain.Invoice to the pdf.InvoiceData struct.
func (h *InvoicePDFHandler) buildPDFData(inv *domain.Invoice) pdf.InvoiceData {
	items := make([]pdf.InvoiceLineItem, len(inv.Items))
	for i, item := range inv.Items {
		productName := item.ProductID.String() // fallback if product not preloaded
		if item.Product.Name != "" {
			productName = item.Product.Name
		}
		items[i] = pdf.InvoiceLineItem{
			SNo:       i + 1,
			Name:      productName,
			HSN:       "", // extend domain.Product with HSN field when needed
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			SubTotal:  item.SubTotal,
			TaxRate:   item.TaxRate,
			TaxAmount: item.TaxAmount,
			Total:     item.Total,
		}
	}

	customerName := fmt.Sprintf("Customer #%d", inv.CustomerID)
	customerGSTIN := ""
	customerAddress := ""
	customerPhone := ""
	if inv.Customer.Name != "" {
		customerName = inv.Customer.Name
		customerGSTIN = inv.Customer.GSTIN
		customerAddress = inv.Customer.Address
		customerPhone = inv.Customer.Phone
	}

	return pdf.InvoiceData{
		SellerName:      h.cfg.GST.SellerName,
		SellerGSTIN:     h.cfg.GST.SellerGSTIN,
		SellerAddress:   h.cfg.GST.SellerAddress,
		SellerState:     h.cfg.GST.StateCode,
		InvoiceNumber:   inv.InvoiceNumber,
		IssuedAt:        inv.IssuedAt,
		DueAt:           inv.DueAt,
		PlaceOfSupply:   customerGSTIN, // first 2 digits = state code
		CustomerName:    customerName,
		CustomerGSTIN:   customerGSTIN,
		CustomerAddress: customerAddress,
		CustomerPhone:   customerPhone,
		Items:           items,
		SubTotal:        inv.SubTotal,
		Discount:        inv.Discount,
		TaxableAmount:   inv.TaxableAmount,
		IntraState:      inv.IntraState,
		CGSTAmount:      inv.CGSTAmount,
		SGSTAmount:      inv.SGSTAmount,
		IGSTAmount:      inv.IGSTAmount,
		TotalAmount:     inv.TotalPrice,
		Notes:           inv.Notes,
	}
}

func (h *InvoicePDFHandler) enqueueInvoiceEmail(
	ctx context.Context,
	inv *domain.Invoice,
	pdfURL, storKey, requestedBy string,
	log *zap.Logger,
) {
	if inv.Customer.Email == "" {
		log.Info("pdf email skipped: customer email unknown")
		return
	}

	emailPayload := EmailPayload{
		To:           inv.Customer.Email,
		Subject:      fmt.Sprintf("Invoice %s from %s", inv.InvoiceNumber, h.cfg.GST.SellerName),
		TemplateName: "invoice_created",
		TemplateData: map[string]any{
			"CustomerName":  inv.Customer.Name,
			"InvoiceNumber": inv.InvoiceNumber,
			"TotalAmount":   fmt.Sprintf("₹%.2f", inv.TotalPrice),
			"DueDate":       inv.DueAt.Format("02 Jan 2006"),
			"CompanyName":   h.cfg.GST.SellerName,
		},
		AttachStorKey: storKey,
		AttachName:    fmt.Sprintf("%s.pdf", inv.InvoiceNumber),
		InvoiceID:     inv.ID.String(),
		RequestedBy:   requestedBy,
	}

	if _, err := h.client.EnqueueEmail(ctx, emailPayload); err != nil {
		// Email failure must not fail the PDF job — log and move on.
		log.Warn("failed to enqueue email after pdf generation", zap.Error(err))
	}
}

// ── error helpers ─────────────────────────────────────────────────────────────

func (h *InvoicePDFHandler) failPermanent(ctx context.Context, taskID string, log *zap.Logger, msg string) error {
	log.Error("pdf job failed permanently", zap.String("reason", msg))
	_ = h.jobRepo.Update(ctx, taskID, repository.UpdateJobParams{
		Status:   domain.JobStatusDead,
		ErrorMsg: msg,
	})
	return fmt.Errorf("%w: %s", asynq.SkipRetry, msg)
}

func (h *InvoicePDFHandler) failRetryable(ctx context.Context, taskID string, log *zap.Logger, err error) error {
	log.Warn("pdf job failed, will retry", zap.Error(err))
	_ = h.jobRepo.Update(ctx, taskID, repository.UpdateJobParams{
		Status:    domain.JobStatusFailed,
		ErrorMsg:  err.Error(),
		IncrRetry: true,
	})
	return err // return without SkipRetry so Asynq retries
}

func isNotFound(err error) bool {
	return errors.Is(err, domain.ErrNotFound)
}
