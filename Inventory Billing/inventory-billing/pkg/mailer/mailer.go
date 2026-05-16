// Package mailer provides a provider-agnostic interface for sending transactional
// emails. The SMTPMailer implementation works with any standard SMTP server
// (Postfix, Gmail, Mailgun SMTP, Resend SMTP bridge, etc.).
package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"runtime"
	"time"

	gomail "gopkg.in/gomail.v2"
)

// Attachment is a file to be attached to an outgoing email.
type Attachment struct {
	Filename    string
	Data        []byte
	ContentType string
}

// Message is a fully constructed email ready to send.
type Message struct {
	To          string
	Subject     string
	HTMLBody    string
	Attachments []Attachment
}

// Mailer sends transactional emails.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// ── SMTP implementation ───────────────────────────────────────────────────────

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string // "Acme Billing <noreply@acme.com>"
	UseTLS   bool   // true = implicit TLS on port 465; false = STARTTLS on 587
}

type smtpMailer struct {
	cfg     SMTPConfig
	dialer  *gomail.Dialer
}

func NewSMTPMailer(cfg SMTPConfig) Mailer {
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	if cfg.UseTLS {
		d.SSL = true
	} else {
		d.TLSConfig = &tls.Config{ServerName: cfg.Host}
	}
	return &smtpMailer{cfg: cfg, dialer: d}
}

func (m *smtpMailer) Send(_ context.Context, msg Message) error {
	gm := gomail.NewMessage()
	gm.SetHeader("From", m.cfg.From)
	gm.SetHeader("To", msg.To)
	gm.SetHeader("Subject", msg.Subject)
	gm.SetBody("text/html", msg.HTMLBody)

	for _, a := range msg.Attachments {
		data := a.Data // capture loop variable
		gm.Attach(a.Filename,
			gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(data)
				return err
			}),
			gomail.SetHeader(map[string][]string{
				"Content-Type": {a.ContentType},
			}),
		)
	}

	if err := m.dialer.DialAndSend(gm); err != nil {
		return fmt.Errorf("smtp: send to %s: %w", msg.To, err)
	}
	return nil
}

// ── no-op mailer for development / testing ────────────────────────────────────

// NoopMailer discards all emails. Use in development to avoid accidental sends.
type NoopMailer struct{}

func (NoopMailer) Send(_ context.Context, msg Message) error {
	return nil // intentional discard
}

// ── HTML template helpers ─────────────────────────────────────────────────────

// templateDir resolves the templates directory relative to this file so the
// caller doesn't need to know the absolute path.
func templateDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "templates")
}

// InvoiceEmailData is the data passed to the invoice email template.
type InvoiceEmailData struct {
	CustomerName  string
	InvoiceNumber string
	TotalAmount   string // pre-formatted with currency symbol
	DueDate       string // pre-formatted
	CompanyName   string
	Year          int
}

// BuildInvoiceEmail renders the HTML body for an invoice-created email.
func BuildInvoiceEmail(data InvoiceEmailData) (string, error) {
	data.Year = time.Now().Year()
	return renderTemplate("invoice_created.html", data)
}

func renderTemplate(name string, data any) (string, error) {
	tmpl, err := template.ParseFiles(filepath.Join(templateDir(), name))
	if err != nil {
		// Fall back to inline template if file is not found (avoids hard dep on filesystem layout).
		return renderInlineInvoiceTemplate(data)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", name, err)
	}
	return buf.String(), nil
}

// renderInlineInvoiceTemplate is the fallback when the template file is missing.
func renderInlineInvoiceTemplate(data any) (string, error) {
	const tpl = `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Invoice</title></head>
<body style="font-family:Arial,sans-serif;color:#333;max-width:600px;margin:0 auto;padding:20px">
  <h2 style="color:#2563eb">{{.CompanyName}}</h2>
  <p>Dear {{.CustomerName}},</p>
  <p>Please find attached your invoice <strong>{{.InvoiceNumber}}</strong> for
     <strong>{{.TotalAmount}}</strong>, due on <strong>{{.DueDate}}</strong>.</p>
  <p>For queries, please reply to this email.</p>
  <p style="color:#888;font-size:12px;margin-top:40px">
    &copy; {{.Year}} {{.CompanyName}}. All rights reserved.
  </p>
</body>
</html>`
	t, err := template.New("invoice").Parse(tpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
