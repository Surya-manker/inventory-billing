package queue

// Task type constants — used as the first argument to asynq.NewTask.
// Format: "<domain>:<verb>" keeps related tasks grouped in the Asynq UI.
const (
	TypeInvoicePDF = "invoice:pdf"
	TypeEmailSend  = "notification:email"

	// Queue names with priority weights (see server.go Config.Queues).
	// critical:default:low = 6:3:1 → workers spend 60% on critical tasks.
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)

// InvoicePDFPayload is marshalled into the Asynq task payload for TypeInvoicePDF.
// Keep it minimal — fetch full data from the DB inside the handler.
type InvoicePDFPayload struct {
	InvoiceID   string `json:"invoice_id"`
	RequestedBy string `json:"requested_by"` // user UUID — for audit trail
}

// InvoicePDFResult is JSON-encoded into JobRecord.Result on success.
type InvoicePDFResult struct {
	PDFURL  string `json:"pdf_url"`  // public URL to download the PDF
	StorKey string `json:"stor_key"` // storage key for internal use
}

// EmailPayload is marshalled into the Asynq task payload for TypeEmailSend.
type EmailPayload struct {
	To           string         `json:"to"`
	Subject      string         `json:"subject"`
	TemplateName string         `json:"template_name"` // e.g. "invoice_created"
	TemplateData map[string]any `json:"template_data"`
	// Attachments are fetched from storage by the worker using the StorKey.
	AttachStorKey string `json:"attach_stor_key,omitempty"`
	AttachName    string `json:"attach_name,omitempty"`    // filename shown to recipient
	InvoiceID     string `json:"invoice_id,omitempty"`     // for entity linking in job_records
	RequestedBy   string `json:"requested_by,omitempty"`
}
