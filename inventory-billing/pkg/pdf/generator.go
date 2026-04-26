// Package pdf generates GST-compliant tax invoice PDFs.
// It wraps the go-pdf/fpdf library behind a Generator interface so the
// underlying library can be swapped (e.g. to chromedp/HTML) without touching
// any business code.
package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

// InvoiceData carries everything the PDF renderer needs. It is populated from
// the domain.Invoice struct by the worker handler.
type InvoiceData struct {
	// Seller (from config)
	SellerName    string
	SellerGSTIN   string
	SellerAddress string
	SellerState   string

	// Invoice header
	InvoiceNumber string
	IssuedAt      time.Time
	DueAt         time.Time
	PlaceOfSupply string // customer state name

	// Customer
	CustomerName    string
	CustomerGSTIN   string
	CustomerAddress string
	CustomerPhone   string

	// Line items
	Items []InvoiceLineItem

	// Totals (pre-calculated by invoice_service)
	SubTotal      float64
	Discount      float64
	TaxableAmount float64
	IntraState    bool
	CGSTAmount    float64
	SGSTAmount    float64
	IGSTAmount    float64
	TotalAmount   float64

	Notes string
}

type InvoiceLineItem struct {
	SNo       int
	Name      string
	HSN       string // HSN/SAC code
	Quantity  int
	UnitPrice float64
	SubTotal  float64
	TaxRate   float64
	TaxAmount float64
	Total     float64
}

// Generator converts an InvoiceData into a PDF byte slice.
type Generator interface {
	Generate(data InvoiceData) ([]byte, error)
}

// ── fpdf implementation ───────────────────────────────────────────────────────

type fpdfGenerator struct{}

func NewGenerator() Generator {
	return &fpdfGenerator{}
}

func (g *fpdfGenerator) Generate(data InvoiceData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	g.drawHeader(pdf, data)
	g.drawParties(pdf, data)
	g.drawItemsTable(pdf, data)
	g.drawTotals(pdf, data)
	g.drawFooter(pdf, data)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf: render: %w", err)
	}
	return buf.Bytes(), nil
}

// ── section renderers ─────────────────────────────────────────────────────────

func (g *fpdfGenerator) drawHeader(pdf *fpdf.Fpdf, d InvoiceData) {
	pageW, _ := pdf.GetPageSize()
	usableW := pageW - 30 // margins 15+15

	// Company name
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(37, 99, 235) // blue-600
	pdf.CellFormat(usableW, 9, d.SellerName, "", 1, "C", false, 0, "")

	// "TAX INVOICE" banner
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(55, 65, 81) // gray-700
	pdf.CellFormat(usableW, 7, "TAX INVOICE", "", 1, "C", false, 0, "")

	pdf.SetDrawColor(37, 99, 235)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, pdf.GetY()+2, pageW-15, pdf.GetY()+2)
	pdf.Ln(5)

	// Seller GSTIN + address on the left, invoice details on the right
	colW := usableW / 2
	startY := pdf.GetY()

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(55, 65, 81)
	pdf.SetXY(15, startY)
	pdf.MultiCell(colW, 5,
		fmt.Sprintf("GSTIN: %s\n%s", d.SellerGSTIN, d.SellerAddress),
		"", "L", false)

	rightY := startY
	pdf.SetXY(15+colW, rightY)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(colW/2, 5, "Invoice No:", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(colW/2, 5, d.InvoiceNumber, "", 1, "L", false, 0, "")

	pdf.SetX(15 + colW)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(colW/2, 5, "Invoice Date:", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(colW/2, 5, d.IssuedAt.Format("02 Jan 2006"), "", 1, "L", false, 0, "")

	pdf.SetX(15 + colW)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(colW/2, 5, "Due Date:", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(colW/2, 5, d.DueAt.Format("02 Jan 2006"), "", 1, "L", false, 0, "")

	pdf.SetX(15 + colW)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(colW/2, 5, "Place of Supply:", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(colW/2, 5, d.PlaceOfSupply, "", 1, "L", false, 0, "")

	// ensure we're below both columns
	if pdf.GetY() < startY+20 {
		pdf.SetY(startY + 20)
	}
	pdf.Ln(4)
}

func (g *fpdfGenerator) drawParties(pdf *fpdf.Fpdf, d InvoiceData) {
	pageW, _ := pdf.GetPageSize()
	usableW := pageW - 30
	colW := usableW / 2
	startY := pdf.GetY()

	pdf.SetFillColor(243, 244, 246) // gray-100
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(55, 65, 81)

	// Headers
	pdf.CellFormat(colW, 6, "  BILLED TO", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colW, 6, "  SHIPPED TO", "1", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetFillColor(255, 255, 255)

	// Customer details (both columns use same data for now)
	leftLines := fmt.Sprintf("%s\n%s\nGSTIN: %s\nPhone: %s",
		d.CustomerName, d.CustomerAddress, d.CustomerGSTIN, d.CustomerPhone)

	currentY := pdf.GetY()
	pdf.SetXY(15, currentY)
	pdf.MultiCell(colW, 5, leftLines, "LRB", "L", false)
	endY := pdf.GetY()

	pdf.SetXY(15+colW, currentY)
	pdf.MultiCell(colW, 5, leftLines, "LRB", "L", false)

	if pdf.GetY() < endY {
		pdf.SetY(endY)
	}
	_ = startY
	pdf.Ln(5)
}

func (g *fpdfGenerator) drawItemsTable(pdf *fpdf.Fpdf, d InvoiceData) {
	pageW, _ := pdf.GetPageSize()
	usableW := pageW - 30

	// Column widths (must sum to usableW = 180)
	cols := []float64{10, 55, 18, 12, 22, 16, 18, 10, 19}
	headers := []string{"#", "Description", "HSN/SAC", "Qty", "Rate (₹)", "SubTotal", "Tax%", "Tax (₹)", "Total (₹)"}

	pdf.SetFillColor(37, 99, 235)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 8)

	for i, h := range headers {
		align := "C"
		if i == 1 {
			align = "L"
		}
		pdf.CellFormat(cols[i], 7, h, "1", 0, align, true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetTextColor(55, 65, 81)
	pdf.SetFont("Arial", "", 8)

	for i, item := range d.Items {
		if i%2 == 0 {
			pdf.SetFillColor(249, 250, 251)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		fill := true

		pdf.CellFormat(cols[0], 6, fmt.Sprintf("%d", item.SNo), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(cols[1], 6, truncate(item.Name, 30), "1", 0, "L", fill, 0, "")
		pdf.CellFormat(cols[2], 6, item.HSN, "1", 0, "C", fill, 0, "")
		pdf.CellFormat(cols[3], 6, fmt.Sprintf("%d", item.Quantity), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(cols[4], 6, fmtAmount(item.UnitPrice), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(cols[5], 6, fmtAmount(item.SubTotal), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(cols[6], 6, fmt.Sprintf("%.1f%%", item.TaxRate), "1", 0, "C", fill, 0, "")
		pdf.CellFormat(cols[7], 6, fmtAmount(item.TaxAmount), "1", 0, "R", fill, 0, "")
		pdf.CellFormat(cols[8], 6, fmtAmount(item.Total), "1", 1, "R", fill, 0, "")
	}

	_ = usableW
	pdf.Ln(3)
}

func (g *fpdfGenerator) drawTotals(pdf *fpdf.Fpdf, d InvoiceData) {
	pageW, _ := pdf.GetPageSize()
	labelW := 60.0
	valueW := 35.0
	x := pageW - 15 - labelW - valueW

	row := func(label string, value float64, bold bool) {
		if bold {
			pdf.SetFont("Arial", "B", 9)
		} else {
			pdf.SetFont("Arial", "", 9)
		}
		pdf.SetX(x)
		pdf.CellFormat(labelW, 6, label, "1", 0, "L", false, 0, "")
		pdf.CellFormat(valueW, 6, fmtAmount(value), "1", 1, "R", false, 0, "")
	}

	pdf.SetTextColor(55, 65, 81)
	row("Sub Total", d.SubTotal, false)
	if d.Discount > 0 {
		row("Discount (-)", d.Discount, false)
		row("Taxable Amount", d.TaxableAmount, false)
	}
	if d.IntraState {
		halfCGST := d.CGSTAmount
		halfSGST := d.SGSTAmount
		row(fmt.Sprintf("CGST (%.1f%%)", halfCGST/d.TaxableAmount*100), halfCGST, false)
		row(fmt.Sprintf("SGST (%.1f%%)", halfSGST/d.TaxableAmount*100), halfSGST, false)
	} else {
		row(fmt.Sprintf("IGST (%.1f%%)", d.IGSTAmount/d.TaxableAmount*100), d.IGSTAmount, false)
	}

	pdf.SetFillColor(37, 99, 235)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetX(x)
	pdf.CellFormat(labelW, 8, "GRAND TOTAL", "1", 0, "L", true, 0, "")
	pdf.CellFormat(valueW, 8, fmtAmount(d.TotalAmount), "1", 1, "R", true, 0, "")

	pdf.SetTextColor(55, 65, 81)
	pdf.Ln(3)
}

func (g *fpdfGenerator) drawFooter(pdf *fpdf.Fpdf, d InvoiceData) {
	pageW, _ := pdf.GetPageSize()
	usableW := pageW - 30

	if d.Notes != "" {
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(usableW, 5, "Notes:", "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "I", 8)
		pdf.MultiCell(usableW, 5, d.Notes, "", "L", false)
		pdf.Ln(3)
	}

	// Signature block
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(107, 114, 128)
	pdf.CellFormat(usableW/2, 5, "This is a computer-generated invoice.", "", 0, "L", false, 0, "")
	pdf.CellFormat(usableW/2, 5, "Authorised Signatory", "", 1, "R", false, 0, "")
	pdf.CellFormat(usableW/2, 5, "", "", 0, "L", false, 0, "")
	pdf.CellFormat(usableW/2, 5, d.SellerName, "", 1, "R", false, 0, "")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func fmtAmount(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}
