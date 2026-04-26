package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-billing/config"
	"github.com/yourusername/inventory-billing/internal/handler"
	"github.com/yourusername/inventory-billing/internal/middleware"
	"github.com/yourusername/inventory-billing/internal/repository"
	"github.com/yourusername/inventory-billing/internal/service"
	"github.com/yourusername/inventory-billing/pkg/cache"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// ── infrastructure ────────────────────────────────────────────────────────
	rdb := cache.NewRedisClient(cfg)
	tokenStore := cache.NewTokenStore(rdb, cfg.JWT.RefreshTokenExpiryDay)

	// ── repositories ─────────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	productRepo := repository.NewProductRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	stockLogRepo := repository.NewStockLogRepository(db)

	// ── services ──────────────────────────────────────────────────────────────
	authSvc := service.NewAuthService(userRepo, tokenStore, cfg.JWT.Secret, cfg.JWT.AccessTokenExpiryHour)
	userSvc := service.NewUserService(userRepo)
	customerSvc := service.NewCustomerService(customerRepo)
	productSvc := service.NewProductService(productRepo)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, productRepo, customerRepo, cfg.GST.StateCode)
	stockSvc := service.NewStockService(productRepo, stockLogRepo)

	// ── handlers ─────────────────────────────────────────────────────────────
	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(userSvc)
	customerH := handler.NewCustomerHandler(customerSvc)
	productH := handler.NewProductHandler(productSvc)
	invoiceH := handler.NewInvoiceHandler(invoiceSvc)
	stockH := handler.NewStockHandler(stockSvc)

	v1 := r.Group("/api/v1")

	// ── public ────────────────────────────────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/refresh", authH.Refresh)
	}

	// ── protected ─────────────────────────────────────────────────────────────
	p := v1.Group("").Use(middleware.Auth(cfg.JWT.Secret))

	// auth self-service
	p.GET("/auth/me", authH.Me)
	p.PUT("/auth/change-password", authH.ChangePassword)
	p.POST("/auth/logout", authH.Logout)

	// users
	p.GET("/users", middleware.RequireRole("admin"), userH.List)
	p.GET("/users/:id", userH.GetByID)
	p.PUT("/users/:id", middleware.RequireRole("admin"), userH.Update)
	p.DELETE("/users/:id", middleware.RequireRole("admin"), userH.Delete)

	// customers
	p.GET("/customers", customerH.List)
	p.GET("/customers/:id", customerH.GetByID)
	p.POST("/customers", middleware.RequireRole("admin"), customerH.Create)
	p.PUT("/customers/:id", middleware.RequireRole("admin"), customerH.Update)
	p.DELETE("/customers/:id", middleware.RequireRole("admin"), customerH.Delete)

	// products
	p.GET("/products", productH.List)
	p.GET("/products/:id", productH.GetByID)
	p.POST("/products", middleware.RequireRole("admin"), productH.Create)
	p.PUT("/products/:id", middleware.RequireRole("admin"), productH.Update)
	p.DELETE("/products/:id", middleware.RequireRole("admin"), productH.Delete)
	p.PATCH("/products/:id/stock", middleware.RequireRole("admin"), productH.AdjustStock)

	// invoices — /number/:number must be declared before /:id
	p.GET("/invoices", invoiceH.List)
	p.POST("/invoices", invoiceH.Create)
	p.GET("/invoices/number/:number", invoiceH.GetByNumber)
	p.GET("/invoices/:id", invoiceH.GetByID)
	p.PATCH("/invoices/:id/status", middleware.RequireRole("admin"), invoiceH.UpdateStatus)
	p.DELETE("/invoices/:id", middleware.RequireRole("admin"), invoiceH.Delete)

	// stock management
	p.GET("/stock/alerts", stockH.GetAlerts)
	p.GET("/stock/logs", middleware.RequireRole("admin"), stockH.GetLogs)
	p.GET("/stock/logs/product/:id", middleware.RequireRole("admin"), stockH.GetProductLogs)

	return r
}
