// cmd/seed/main.go
// Run with:  go run ./cmd/seed
//
// Safe to run multiple times — every record is skipped if it already exists.
// Insertion order: users → customers → products → invoices (with items + stock logs).
package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/inventory-billing/config"
	"github.com/yourusername/inventory-billing/internal/domain"
	"github.com/yourusername/inventory-billing/pkg/database"
	"github.com/yourusername/inventory-billing/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	log.Println("▶ seeding users...")
	users := seedUsers(db)

	log.Println("▶ seeding customers...")
	customers := seedCustomers(db)

	log.Println("▶ seeding products...")
	products := seedProducts(db)

	log.Println("▶ seeding invoices...")
	seedInvoices(db, users, customers, products, cfg.GST.StateCode)

	log.Println("✓ seed complete")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// mustHash panics on bcrypt failure — only acceptable in a seed script.
func mustHash(plain string) string {
	h, err := utils.HashPassword(plain)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	return h
}

// upsertUser inserts the user only when the email is not already taken.
func upsertUser(db *gorm.DB, u *domain.User) *domain.User {
	var existing domain.User
	if err := db.Where("email = ?", u.Email).First(&existing).Error; err == nil {
		log.Printf("   skip user %s (already exists)", u.Email)
		return &existing
	}
	if err := db.Create(u).Error; err != nil {
		log.Fatalf("create user %s: %v", u.Email, err)
	}
	log.Printf("   + user  %-30s  role=%s", u.Email, u.Role)
	return u
}

// upsertCustomer inserts the customer only when the email is not already taken.
func upsertCustomer(db *gorm.DB, c *domain.Customer) *domain.Customer {
	var existing domain.Customer
	if err := db.Where("email = ?", c.Email).First(&existing).Error; err == nil {
		log.Printf("   skip customer %s (already exists)", c.Email)
		return &existing
	}
	if err := db.Create(c).Error; err != nil {
		log.Fatalf("create customer %s: %v", c.Email, err)
	}
	log.Printf("   + customer %-30s  gstin=%s", c.Email, c.GSTIN)
	return c
}

// upsertProduct inserts the product only when the SKU is not already taken.
func upsertProduct(db *gorm.DB, p *domain.Product) *domain.Product {
	var existing domain.Product
	if err := db.Where("sku = ?", p.SKU).First(&existing).Error; err == nil {
		log.Printf("   skip product %s (already exists)", p.SKU)
		return &existing
	}
	if err := db.Create(p).Error; err != nil {
		log.Fatalf("create product %s: %v", p.SKU, err)
	}
	log.Printf("   + product %-12s  ₹%-8.2f  stock=%-4d  tax=%.0f%%", p.SKU, p.Price, p.Stock, p.TaxRate)
	return p
}

// nextInvoiceNumber inserts one row into invoice_counters and returns the
// formatted number — exactly the same logic as invoice_repository.CreateTx.
func nextInvoiceNumber(tx *gorm.DB) string {
	counter := domain.InvoiceCounter{}
	if err := tx.Create(&counter).Error; err != nil {
		log.Fatalf("invoice counter: %v", err)
	}
	return fmt.Sprintf("INV-%s-%07d", time.Now().UTC().Format("200601"), counter.ID)
}

// isIntraState mirrors utils.IsIntraState without importing the package.
func isIntraState(sellerStateCode, gstin string) bool {
	if sellerStateCode == "" || len(gstin) < 2 {
		return true
	}
	return gstin[:2] == sellerStateCode
}

// ── users ─────────────────────────────────────────────────────────────────────

func seedUsers(db *gorm.DB) map[string]*domain.User {
	rows := []domain.User{
		{
			Name:     "Hemant Sharma",
			Email:    "hemant@adpixel.tech",
			Password: mustHash("Admin@123"),
			Role:     domain.RoleAdmin,
		},
		{
			Name:     "Priya Patel",
			Email:    "priya@adpixel.tech",
			Password: mustHash("Staff@123"),
			Role:     domain.RoleStaff,
		},
		{
			Name:     "Arjun Mehta",
			Email:    "arjun@adpixel.tech",
			Password: mustHash("Staff@123"),
			Role:     domain.RoleStaff,
		},
	}

	out := make(map[string]*domain.User, len(rows))
	for i := range rows {
		out[rows[i].Email] = upsertUser(db, &rows[i])
	}
	return out
}

// ── customers ─────────────────────────────────────────────────────────────────

func seedCustomers(db *gorm.DB) map[string]*domain.Customer {
	// GSTINs: first 2 digits = state code.
	// Seller is Karnataka (29).  Customers in 29 → intra-state; others → inter-state.
	rows := []domain.Customer{
		{
			Name:    "Ravi Electronics",
			Email:   "ravi@ravielectronics.com",
			Phone:   "+91-9876543210",
			Address: "42 MG Road, Bengaluru, Karnataka 560001",
			Company: "Ravi Electronics Pvt Ltd",
			GSTIN:   "29AAACR1234F1Z5", // state 29 → intra-state
		},
		{
			Name:    "Sharma Traders",
			Email:   "billing@sharmatraders.com",
			Phone:   "+91-9823456789",
			Address: "15 Nehru Street, Mumbai, Maharashtra 400001",
			Company: "Sharma Traders",
			GSTIN:   "27AAGCS5678K2Y6", // state 27 → inter-state
		},
		{
			Name:    "Kumar Office Supplies",
			Email:   "accounts@kumaroffice.com",
			Phone:   "+91-9712345678",
			Address: "8 Park Avenue, Connaught Place, New Delhi 110001",
			Company: "Kumar Office Supplies LLP",
			GSTIN:   "07AAFCK9012L3X7", // state 07 → inter-state
		},
		{
			Name:    "Gupta Wholesale",
			Email:   "gupta@guptawholesale.com",
			Phone:   "+91-9651234567",
			Address: "23 Gandhi Road, Kolkata, West Bengal 700001",
			Company: "Gupta Wholesale Distributors",
			GSTIN:   "19AABCG3456M4W8", // state 19 → inter-state
		},
		{
			Name:    "Singh & Sons",
			Email:   "finance@singhsons.com",
			Phone:   "+91-9540123456",
			Address: "67 Civil Lines, Lucknow, Uttar Pradesh 226001",
			Company: "Singh & Sons Enterprises",
			GSTIN:   "09AADCS7890N5V9", // state 09 → inter-state
		},
	}

	out := make(map[string]*domain.Customer, len(rows))
	for i := range rows {
		out[rows[i].Email] = upsertCustomer(db, &rows[i])
	}
	return out
}

// ── products ──────────────────────────────────────────────────────────────────

func seedProducts(db *gorm.DB) map[string]*domain.Product {
	// GST slabs used in India:
	//   5%  — essential goods
	//  12%  — standard stationery, lamps
	//  18%  — electronics, cables, hubs
	//  28%  — luxury goods (none here)
	rows := []domain.Product{
		{
			Name:              "USB-C Charging Cable 2m",
			Description:       "Braided nylon USB-C to USB-C cable, 100W fast charging, 2 metre",
			SKU:               "CBL-USBC-2M",
			Price:             299.00,
			Stock:             150,
			TaxRate:           18,
			LowStockThreshold: 20,
		},
		{
			Name:              "Wireless Optical Mouse",
			Description:       "2.4 GHz wireless mouse, 1600 DPI, USB nano receiver, 12-month battery life",
			SKU:               "MSE-WL-001",
			Price:             899.00,
			Stock:             75,
			TaxRate:           18,
			LowStockThreshold: 10,
		},
		{
			Name:              "HDMI Cable 2m v2.0",
			Description:       "High-speed HDMI 2.0 cable, supports 4K@60Hz, gold-plated connectors",
			SKU:               "CBL-HDMI-2M",
			Price:             449.00,
			Stock:             100,
			TaxRate:           18,
			LowStockThreshold: 15,
		},
		{
			Name:              "A4 Paper Ream 75gsm",
			Description:       "500 sheets, 75 GSM, acid-free, suitable for laser and inkjet printers",
			SKU:               "PPR-A4-75G",
			Price:             350.00,
			Stock:             200,
			TaxRate:           12,
			LowStockThreshold: 30,
		},
		{
			Name:              "Ball Pen Pack (10 pcs)",
			Description:       "Blue ink ball pens, medium tip 0.7mm, smooth writing, refillable",
			SKU:               "PEN-BALL-10",
			Price:             85.00,
			Stock:             500,
			TaxRate:           12,
			LowStockThreshold: 50,
		},
		{
			Name:              "Adjustable Laptop Stand",
			Description:       "Aluminium alloy, 6 height settings, foldable, fits laptops up to 17 inches",
			SKU:               "STD-LAP-ALU",
			Price:             1499.00,
			Stock:             40,
			TaxRate:           18,
			LowStockThreshold: 5,
		},
		{
			Name:              "LED Desk Lamp with USB Port",
			Description:       "10W LED, 5 colour temperatures, touch dimmer, built-in USB-A charging port",
			SKU:               "LMP-LED-USB",
			Price:             799.00,
			Stock:             60,
			TaxRate:           12,
			LowStockThreshold: 10,
		},
		{
			Name:              "Heavy Duty Stapler",
			Description:       "Staples up to 50 sheets, includes 1000 staples, ergonomic soft-grip handle",
			SKU:               "STP-HD-50",
			Price:             199.00,
			Stock:             120,
			TaxRate:           12,
			LowStockThreshold: 15,
		},
		{
			Name:              "Sticky Notes 76x76mm (5 pads)",
			Description:       "100 sheets per pad, assorted neon colours, repositionable adhesive",
			SKU:               "STK-76-5PK",
			Price:             129.00,
			Stock:             300,
			TaxRate:           12,
			LowStockThreshold: 40,
		},
		{
			Name:              "USB 3.0 Hub 4-Port",
			Description:       "USB 3.0 superspeed hub, 5 Gbps data transfer, individual LED indicators",
			SKU:               "HUB-USB3-4P",
			Price:             649.00,
			Stock:             8,  // intentionally low → triggers stock alert
			TaxRate:           18,
			LowStockThreshold: 10,
		},
	}

	out := make(map[string]*domain.Product, len(rows))
	for i := range rows {
		out[rows[i].SKU] = upsertProduct(db, &rows[i])
	}
	return out
}

// ── invoices ──────────────────────────────────────────────────────────────────

type lineInput struct {
	product  *domain.Product
	quantity int
}

// buildInvoice computes every financial field from the line items — mirrors
// invoice_service.go so the seeded values are identical to what the real API
// would produce.
func buildInvoice(
	db *gorm.DB,
	customer *domain.Customer,
	createdBy uuid.UUID,
	lines []lineInput,
	discount float64,
	status domain.InvoiceStatus,
	notes string,
	issuedAt, dueAt time.Time,
	sellerStateCode string,
) *domain.Invoice {
	intra := isIntraState(sellerStateCode, customer.GSTIN)

	var items []domain.InvoiceItem
	var subTotal, taxAmount float64

	for _, l := range lines {
		lineSub := round2(l.product.Price * float64(l.quantity))
		lineTax := round2(lineSub * l.product.TaxRate / 100)

		items = append(items, domain.InvoiceItem{
			ProductID: l.product.ID,
			Quantity:  l.quantity,
			UnitPrice: l.product.Price,
			SubTotal:  lineSub,
			TaxRate:   l.product.TaxRate,
			TaxAmount: lineTax,
			Total:     round2(lineSub + lineTax),
		})
		subTotal += lineSub
		taxAmount += lineTax
	}

	subTotal    = round2(subTotal)
	taxAmount   = round2(taxAmount)
	taxable     := round2(subTotal - discount)
	totalPrice  := round2(taxable + taxAmount)

	// GST split — same as utils.SplitGST
	var cgst, sgst, igst float64
	if intra {
		half := round2(taxAmount / 2)
		cgst  = half
		sgst  = round2(taxAmount - half)
	} else {
		igst = taxAmount
	}

	inv := &domain.Invoice{
		CustomerID:    customer.ID,
		CreatedBy:     createdBy,
		Items:         items,
		SubTotal:      subTotal,
		Discount:      discount,
		TaxableAmount: taxable,
		TaxAmount:     taxAmount,
		TotalPrice:    totalPrice,
		IntraState:    intra,
		CGSTAmount:    cgst,
		SGSTAmount:    sgst,
		IGSTAmount:    igst,
		Status:        status,
		Notes:         notes,
		IssuedAt:      issuedAt,
		DueAt:         dueAt,
	}

	// Wrap in a transaction so invoice_number, invoice rows, stock deductions,
	// and stock_logs are all committed atomically or not at all.
	err := db.Transaction(func(tx *gorm.DB) error {
		inv.InvoiceNumber = nextInvoiceNumber(tx)

		// Lock products in deterministic order (same deadlock-prevention technique
		// as the real invoice_repository.CreateTx).
		pids := make([]uuid.UUID, 0, len(lines))
		seen := map[uuid.UUID]bool{}
		for _, l := range lines {
			if !seen[l.product.ID] {
				pids = append(pids, l.product.ID)
				seen[l.product.ID] = true
			}
		}

		var locked []domain.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", pids).Find(&locked).Error; err != nil {
			return err
		}
		stockMap := map[uuid.UUID]int{}
		for _, p := range locked {
			stockMap[p.ID] = p.Stock
		}

		if err := tx.Create(inv).Error; err != nil {
			return err
		}

		// Only deduct stock for pending/paid invoices.
		// Canceled invoices have stock already "restored" (we seed them in net state).
		if status != domain.InvoiceCanceled {
			for _, l := range lines {
				before := stockMap[l.product.ID]
				after  := before - l.quantity

				if err := tx.Model(&domain.Product{}).
					Where("id = ?", l.product.ID).
					UpdateColumn("stock", after).Error; err != nil {
					return err
				}

				sl := domain.StockLog{
					ProductID:      l.product.ID,
					UserID:         createdBy,
					InvoiceID:      &inv.ID,
					ChangeType:     domain.StockSale,
					QuantityBefore: before,
					QuantityChange: -l.quantity,
					QuantityAfter:  after,
					Note:           fmt.Sprintf("sold via invoice %s", inv.InvoiceNumber),
				}
				if err := tx.Create(&sl).Error; err != nil {
					return err
				}
				stockMap[l.product.ID] = after
			}
		}

		if status == domain.InvoicePaid {
			now := time.Now().UTC()
			inv.PaidAt = &now
			if err := tx.Model(inv).Updates(map[string]any{
				"status":  domain.InvoicePaid,
				"paid_at": now,
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		log.Fatalf("seed invoice for customer %s: %v", customer.Name, err)
	}

	log.Printf("   + invoice %-20s  customer=%-22s  total=₹%-10.2f  status=%s",
		inv.InvoiceNumber, customer.Name, inv.TotalPrice, inv.Status)
	return inv
}

func seedInvoices(
	db *gorm.DB,
	users map[string]*domain.User,
	customers map[string]*domain.Customer,
	products map[string]*domain.Product,
	sellerStateCode string,
) {
	// Skip seeding if any invoices already exist.
	var count int64
	db.Model(&domain.Invoice{}).Count(&count)
	if count > 0 {
		log.Printf("   skip invoices (%d already exist)", count)
		return
	}

	admin := users["hemant@adpixel.tech"]
	staff := users["priya@adpixel.tech"]

	now := time.Now().UTC()

	// ── Invoice 1: Ravi Electronics — PAID — intra-state (both Karnataka) ─────
	// USB-C Cable ×3, Wireless Mouse ×2 | no discount
	buildInvoice(db, customers["ravi@ravielectronics.com"], admin.ID,
		[]lineInput{
			{products["CBL-USBC-2M"], 3},
			{products["MSE-WL-001"], 2},
		},
		0,                          // discount
		domain.InvoicePaid,
		"Bulk order for office setup. Delivered and verified.",
		now.AddDate(0, -1, 0),      // issued 1 month ago
		now.AddDate(0, -1, 15),     // due 15 days after issue
		sellerStateCode,
	)

	// ── Invoice 2: Sharma Traders — PENDING — inter-state (Maharashtra) ───────
	// A4 Paper ×5, Ball Pen Pack ×10 | ₹100 discount
	buildInvoice(db, customers["billing@sharmatraders.com"], staff.ID,
		[]lineInput{
			{products["PPR-A4-75G"], 5},
			{products["PEN-BALL-10"], 10},
		},
		100,                        // discount
		domain.InvoicePending,
		"Monthly stationery order. Payment expected within 15 days.",
		now.AddDate(0, 0, -10),     // issued 10 days ago
		now.AddDate(0, 0, 20),      // due in 20 days
		sellerStateCode,
	)

	// ── Invoice 3: Kumar Office Supplies — CANCELED — inter-state (Delhi) ─────
	// Laptop Stand ×1, LED Lamp ×2 | no discount | stock NOT deducted (canceled)
	buildInvoice(db, customers["accounts@kumaroffice.com"], admin.ID,
		[]lineInput{
			{products["STD-LAP-ALU"], 1},
			{products["LMP-LED-USB"], 2},
		},
		0,
		domain.InvoiceCanceled,
		"Order canceled by customer — product no longer required.",
		now.AddDate(0, 0, -20),     // issued 20 days ago
		now.AddDate(0, 0, 10),      // due date (never reached)
		sellerStateCode,
	)

	// ── Invoice 4: Ravi Electronics — PENDING — intra-state ──────────────────
	// USB Hub ×2 | no discount
	buildInvoice(db, customers["ravi@ravielectronics.com"], staff.ID,
		[]lineInput{
			{products["HUB-USB3-4P"], 2},
		},
		0,
		domain.InvoicePending,
		"Urgent requirement for server room expansion.",
		now.AddDate(0, 0, -3),      // issued 3 days ago
		now.AddDate(0, 0, 27),      // due in 27 days
		sellerStateCode,
	)

	// ── Invoice 5: Gupta Wholesale — PAID — inter-state (West Bengal) ─────────
	// Stapler ×3, Sticky Notes ×10 | no discount
	buildInvoice(db, customers["gupta@guptawholesale.com"], admin.ID,
		[]lineInput{
			{products["STP-HD-50"], 3},
			{products["STK-76-5PK"], 10},
		},
		0,
		domain.InvoicePaid,
		"Quarterly stationery replenishment. Payment received via NEFT.",
		now.AddDate(0, 0, -7),      // issued 7 days ago
		now.AddDate(0, 0, 23),      // due in 23 days
		sellerStateCode,
	)

	// ── Invoice 6: Singh & Sons — PENDING — inter-state (Uttar Pradesh) ───────
	// HDMI Cable ×5, Wireless Mouse ×1 | ₹200 discount
	buildInvoice(db, customers["finance@singhsons.com"], staff.ID,
		[]lineInput{
			{products["CBL-HDMI-2M"], 5},
			{products["MSE-WL-001"], 1},
		},
		200,                        // discount
		domain.InvoicePending,
		"New branch setup — AV and peripherals.",
		now.AddDate(0, 0, -1),      // issued yesterday
		now.AddDate(0, 1, 0),       // due in 1 month
		sellerStateCode,
	)

	// ── Manual stock adjustment: received a new shipment of USB Hubs ──────────
	// This shows a non-invoice stock movement in stock_logs.
	hub := products["HUB-USB3-4P"]
	var currentHub domain.Product
	db.First(&currentHub, "id = ?", hub.ID)

	adjErr := db.Transaction(func(tx *gorm.DB) error {
		before := currentHub.Stock
		after  := before + 25 // restocked

		if err := tx.Model(&domain.Product{}).
			Where("id = ?", hub.ID).
			UpdateColumn("stock", after).Error; err != nil {
			return err
		}
		sl := domain.StockLog{
			ProductID:      hub.ID,
			UserID:         admin.ID,
			ChangeType:     domain.StockPurchase,
			QuantityBefore: before,
			QuantityChange: 25,
			QuantityAfter:  after,
			Note:           "Restocked from supplier — PO#2024-087",
		}
		return tx.Create(&sl).Error
	})
	if adjErr != nil {
		log.Fatalf("stock adjustment: %v", adjErr)
	}
	log.Printf("   + stock_log  HUB-USB3-4P  purchase +25  (low-stock restocked)")
}
