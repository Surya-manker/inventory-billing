package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/config"
	"github.com/yourusername/inventory-billing/internal/handler"
	"github.com/yourusername/inventory-billing/internal/middleware"
	"github.com/yourusername/inventory-billing/internal/repository"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/cache"
	"github.com/yourusername/inventory-billing/pkg/queue"
	"github.com/yourusername/inventory-billing/pkg/storage"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	log, _ := zap.NewProduction()

	r := gin.New()
	r.Use(
		gin.Recovery(),
		middleware.Logger(log),
		middleware.RateLimiter(20, 50),
		middleware.AuditLog(db), // append-only audit trail for all mutations
	)

	// ── health checks (no auth required) ─────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/health/ready", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db_down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// ── static assets (invoice PDFs stored locally) ───────────────────────────
	if cfg.Storage.Provider == "local" {
		r.Static("/assets", cfg.Storage.LocalDir)
	}

	// ── infrastructure ────────────────────────────────────────────────────────
	rdb := cache.NewRedisClient(cfg)
	tokenStore := cache.NewTokenStore(rdb, cfg.JWT.RefreshTokenExpiryDay)
	genericCache := cache.NewCache(rdb)

	// ── job queue client ──────────────────────────────────────────────────────
	jobRepo := repository.NewJobRepository(db)
	queueClient := queue.NewClient(cfg, jobRepo, log)

	// ── storage backend ───────────────────────────────────────────────────────
	var stor storage.Storage
	if cfg.Storage.Provider == "local" {
		var err error
		stor, err = storage.NewLocalStorage(cfg.Storage.LocalDir, cfg.Storage.LocalURL)
		if err != nil {
			log.Fatal("failed to init local storage", zap.Error(err))
		}
	}
	_ = stor // used by worker; server only needs it for health checks

	// ── repositories ─────────────────────────────────────────────────────────
	userRepo       := repository.NewUserRepository(db)
	customerRepo   := repository.NewCustomerRepository(db)
	productRepo    := repository.NewProductRepository(db)
	invoiceRepo    := repository.NewInvoiceRepository(db)
	stockLogRepo   := repository.NewStockLogRepository(db)
	vendorRepo     := repository.NewVendorRepository(db)
	purchaseRepo   := repository.NewPurchaseRepository(db)
	paymentRepo    := repository.NewPaymentRepository(db)
	reportRepo     := repository.NewReportRepository(db)
	categoryRepo   := repository.NewCategoryRepository(db)
	creditNoteRepo := repository.NewCreditNoteRepository(db)

	// ── services ──────────────────────────────────────────────────────────────
	authSvc       := service.NewAuthService(userRepo, tokenStore, cfg.JWT.Secret, cfg.JWT.AccessTokenExpiryHour)
	userSvc       := service.NewUserService(userRepo)
	customerSvc   := service.NewCustomerService(customerRepo)
	productSvc    := service.NewProductService(productRepo, genericCache)  // cache enabled
	invoiceSvc    := service.NewInvoiceService(invoiceRepo, productRepo, customerRepo, cfg.GST.StateCode)
	stockSvc      := service.NewStockService(productRepo, stockLogRepo)
	vendorSvc     := service.NewVendorService(vendorRepo)
	purchaseSvc   := service.NewPurchaseService(purchaseRepo, vendorRepo)
	paymentSvc    := service.NewPaymentService(paymentRepo, invoiceRepo)
	reportSvc     := service.NewReportService(reportRepo)
	categorySvc   := service.NewCategoryService(categoryRepo)
	creditNoteSvc := service.NewCreditNoteService(creditNoteRepo, invoiceRepo)

	// ── handlers ─────────────────────────────────────────────────────────────
	authH         := handler.NewAuthHandler(authSvc)
	userH         := handler.NewUserHandler(userSvc)
	customerH     := handler.NewCustomerHandler(customerSvc)
	productH      := handler.NewProductHandler(productSvc)
	jobH          := handler.NewJobHandler(jobRepo)
	invoiceH      := handler.NewInvoiceHandler(invoiceSvc, jobRepo, queueClient)
	stockH        := handler.NewStockHandler(stockSvc)
	vendorH       := handler.NewVendorHandler(vendorSvc)
	purchaseH     := handler.NewPurchaseHandler(purchaseSvc)
	paymentH      := handler.NewPaymentHandler(paymentSvc)
	reportH       := handler.NewReportHandler(reportSvc)
	categoryH     := handler.NewCategoryHandler(categorySvc)
	creditNoteH   := handler.NewCreditNoteHandler(creditNoteSvc)

	v1 := r.Group("/api/v1")

	// ── public ────────────────────────────────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login",    authH.Login)
		auth.POST("/refresh",  authH.Refresh)
	}

	// ── protected — requires valid Bearer token ───────────────────────────────
	p     := v1.Group("", middleware.Auth(cfg.JWT.Secret))
	// admin — requires Bearer token + admin role
	admin := v1.Group("", middleware.Auth(cfg.JWT.Secret), middleware.RequireRole("admin"))

	// auth self-service
	p.GET("/auth/me",              authH.Me)
	p.PUT("/auth/change-password", authH.ChangePassword)
	p.POST("/auth/logout",         authH.Logout)

	// users
	admin.GET("/users",       userH.List)
	p.GET("/users/:id",       userH.GetByID)
	admin.PUT("/users/:id",   userH.Update)
	admin.DELETE("/users/:id", userH.Delete)

	// customers
	p.GET("/customers",         customerH.List)
	p.GET("/customers/:id",     customerH.GetByID)
	admin.POST("/customers",    customerH.Create)
	admin.PUT("/customers/:id", customerH.Update)
	admin.DELETE("/customers/:id", customerH.Delete)

	// products
	p.GET("/products",              productH.List)
	p.GET("/products/:id",          productH.GetByID)
	admin.POST("/products",         productH.Create)
	admin.PUT("/products/:id",      productH.Update)
	admin.DELETE("/products/:id",   productH.Delete)
	admin.PATCH("/products/:id/stock", productH.AdjustStock)

	// job status — generic polling endpoint for any async operation
	p.GET("/jobs/:id", jobH.GetByID)

	// invoices — /number/:number must be declared before /:id
	p.GET("/invoices",                invoiceH.List)
	p.POST("/invoices",               invoiceH.Create)
	p.GET("/invoices/number/:number", invoiceH.GetByNumber)
	p.GET("/invoices/:id",            invoiceH.GetByID)
	p.GET("/invoices/:id/pdf",        invoiceH.GetPDFStatus) // 202 while pending, 200 when ready
	admin.PATCH("/invoices/:id/status", invoiceH.UpdateStatus)
	admin.DELETE("/invoices/:id",       invoiceH.Delete)

	// payments
	p.GET("/invoices/:id/payments",    paymentH.ListByInvoice)
	p.POST("/invoices/:id/payments",   paymentH.Record)
	admin.DELETE("/payments/:id",      paymentH.Delete)

	// stock management
	p.GET("/stock/alerts",                 stockH.GetAlerts)
	admin.GET("/stock/logs",               stockH.GetLogs)
	admin.GET("/stock/logs/product/:id",   stockH.GetProductLogs)

	// vendors — admin only
	admin.GET("/vendors",            vendorH.List)
	admin.POST("/vendors",           vendorH.Create)
	admin.GET("/vendors/:id",        vendorH.GetByID)
	admin.PUT("/vendors/:id",        vendorH.Update)
	admin.PATCH("/vendors/:id/deactivate", vendorH.Deactivate)
	admin.DELETE("/vendors/:id",     vendorH.Delete)

	// purchase orders — admin only
	admin.GET("/purchase-orders",              purchaseH.List)
	admin.POST("/purchase-orders",             purchaseH.Create)
	admin.GET("/purchase-orders/:id",          purchaseH.GetByID)
	admin.PATCH("/purchase-orders/:id/status", purchaseH.UpdateStatus)
	admin.DELETE("/purchase-orders/:id",       purchaseH.Delete)

	// reports — admin only
	admin.GET("/reports/revenue",         reportH.Revenue)
	admin.GET("/reports/top-products",    reportH.TopProducts)
	admin.GET("/reports/invoices",        reportH.InvoiceSummary)
	admin.GET("/reports/inventory-value", reportH.InventoryValue)
	admin.GET("/reports/aging",           reportH.AgingReport)

	// categories — admin write, any auth read
	p.GET("/categories",        categoryH.List)
	p.GET("/categories/:id",    categoryH.GetByID)
	admin.POST("/categories",   categoryH.Create)
	admin.PUT("/categories/:id", categoryH.Update)
	admin.DELETE("/categories/:id", categoryH.Delete)

	// credit notes
	admin.POST("/invoices/:id/credit-notes",     creditNoteH.Issue)
	p.GET("/invoices/:id/credit-notes",          creditNoteH.ListByInvoice)
	p.GET("/credit-notes/:id",                   creditNoteH.GetByID)
	admin.PATCH("/credit-notes/:id/void",        creditNoteH.Void)
	p.GET("/customers/:id/credit-notes",         creditNoteH.ListByCustomer)

	return r
}
