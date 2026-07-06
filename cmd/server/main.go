package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/ArthurWerle/transactions/internal/config"
	"github.com/ArthurWerle/transactions/internal/handler"
	"github.com/ArthurWerle/transactions/internal/migrations"
	"github.com/ArthurWerle/transactions/internal/repository"
	"github.com/ArthurWerle/transactions/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	logger := setupLogger(cfg.Log.Level)
	logger.Info("starting transaction service")

	db, err := setupDatabase(cfg, logger)
	if err != nil {
		logger.Error("failed to setup database", "error", err)
		os.Exit(1)
	}

	// The versioned SQL migrations are the only schema owner.
	if err := migrations.RunMigrations(db, logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	reportingLoc, err := time.LoadLocation(cfg.Reporting.Timezone)
	if err != nil {
		logger.Error("invalid REPORTING_TIMEZONE", "timezone", cfg.Reporting.Timezone, "error", err)
		os.Exit(1)
	}

	locationRepo := repository.NewLocationRepository(db)
	locationService := service.NewLocationService(locationRepo)
	locationHandler := handler.NewLocationHandler(locationService)

	var identityClient service.IdentityClient
	if cfg.Identity.BaseURL != "" {
		identityClient = service.NewHTTPIdentityClient(cfg.Identity.BaseURL)
	}

	transactionRepo := repository.NewTransactionsRepository(db, reportingLoc)
	transactionService := service.NewTransactionsService(transactionRepo, reportingLoc)
	transactionHandler := handler.NewTransactionHandler(transactionService, locationService, identityClient, reportingLoc)

	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := service.NewCategoryService(categoryRepo)
	categoryHandler := handler.NewCategoryHandler(categoryService)

	subcategoryRepo := repository.NewSubcategoryRepository(db)
	subcategoryService := service.NewSubcategoryService(subcategoryRepo)
	subcategoryHandler := handler.NewSubcategoryHandler(subcategoryService)

	router := setupRouter(cfg, logger, transactionHandler, categoryHandler, subcategoryHandler, locationHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: router,
	}

	go func() {
		logger.Info("starting HTTP server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server exited")
}

func setupRouter(cfg *config.Config, logger *slog.Logger, transactionHandler *handler.TransactionHandler, categoryHandler *handler.CategoryHandler, subcategoryHandler *handler.SubcategoryHandler, locationHandler *handler.LocationHandler) *gin.Engine {
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	api := router.Group("/api/v2")
	{
		transactions := api.Group("/transactions")
		{
			transactions.GET("", transactionHandler.GetTransactions)
			transactions.POST("", transactionHandler.CreateTransaction)
			transactions.GET("/:id", transactionHandler.GetTransactionByID)
			transactions.PUT("/:id", transactionHandler.UpdateTransaction)
			transactions.DELETE("/:id", transactionHandler.DeleteTransaction)
			transactions.POST("/:id/prepay", transactionHandler.PrepayTransaction)
			transactions.PATCH("/:id/end", transactionHandler.EndRecurringTransaction)
			transactions.GET("/by-date-range", transactionHandler.GetTransactionsByDateRange)
			transactions.GET("/latest", transactionHandler.GetLatestTransactions)
			transactions.GET("/biggest", transactionHandler.GetBiggestTransactions)
			transactions.GET("/average/by-type", transactionHandler.GetAverageByType)
			transactions.GET("/average/by-category", transactionHandler.GetAverageByCategory)

			reports := transactions.Group("/reports")
			{
				reports.GET("/monthly-history", transactionHandler.GetMonthlyHistory)
				reports.GET("/category-history", transactionHandler.GetCategoryHistory)
				reports.GET("/month-overview", transactionHandler.GetMonthOverview)
				reports.GET("/monthly-expenses-by-category", transactionHandler.GetMonthlyExpensesByCategory)
			}
		}

		categories := api.Group("/categories")
		{
			categories.GET("", categoryHandler.GetCategories)
			categories.POST("", categoryHandler.CreateCategory)
			categories.GET("/:id", categoryHandler.GetCategoryByID)
			categories.PUT("/:id", categoryHandler.UpdateCategory)
			categories.DELETE("/:id", categoryHandler.DeleteCategory)
		}

		subcategories := api.Group("/subcategories")
		{
			subcategories.GET("", subcategoryHandler.GetSubcategories)
			subcategories.POST("", subcategoryHandler.CreateSubcategory)
			subcategories.GET("/:id", subcategoryHandler.GetSubcategoryByID)
			subcategories.PUT("/:id", subcategoryHandler.UpdateSubcategory)
			subcategories.DELETE("/:id", subcategoryHandler.DeleteSubcategory)
		}

		locations := api.Group("/locations")
		{
			locations.GET("", locationHandler.GetLocations)
			locations.POST("", locationHandler.CreateLocation)
			locations.POST("/merge", locationHandler.MergeLocations)
			locations.GET("/:id", locationHandler.GetLocationByID)
			locations.PUT("/:id", locationHandler.UpdateLocation)
			locations.DELETE("/:id", locationHandler.DeleteLocation)
		}
	}

	return router
}

func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

func setupDatabase(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	gormLogLevel := gormLogger.Silent
	if cfg.Log.Level == "debug" {
		gormLogLevel = gormLogger.Info
	}

	gormCfg := &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogLevel),
		// Translate driver errors (e.g. unique violations) into gorm.Err*
		// sentinels so handlers can map them to proper HTTP statuses.
		TranslateError: true,
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Info("database connection established")
	return db, nil
}
