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

	"github.com/ArthurWerle/transactions/internal/config"
	"github.com/ArthurWerle/transactions/internal/handler"
	"github.com/ArthurWerle/transactions/internal/migrations"
	"github.com/ArthurWerle/transactions/internal/model"
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

	// Run SQL migrations
	if err := migrations.RunMigrations(db, logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	if err := db.AutoMigrate(
		&model.Transaction{},
		&model.Category{},
		&model.Subcategory{},
	); err != nil {
		logger.Error("failed to auto migrate", "error", err)
		os.Exit(1)
	}

	transactionRepo := repository.NewTransactionsRepository(db)
	transactionService := service.NewTransactionsService(transactionRepo)
	transactionHandler := handler.NewTransactionHandler(transactionService)

	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := service.NewCategoryService(categoryRepo)
	categoryHandler := handler.NewCategoryHandler(categoryService)

	subcategoryRepo := repository.NewSubcategoryRepository(db)
	subcategoryService := service.NewSubcategoryService(subcategoryRepo)
	subcategoryHandler := handler.NewSubcategoryHandler(subcategoryService)

	router := setupRouter(cfg, logger, transactionHandler, categoryHandler, subcategoryHandler)

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

func setupRouter(cfg *config.Config, logger *slog.Logger, transactionHandler *handler.TransactionHandler, categoryHandler *handler.CategoryHandler, subcategoryHandler *handler.SubcategoryHandler) *gin.Engine {
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

	v1 := router.Group("/api/v2")
	{
		transactions := v1.Group("/transactions")
		{
			transactions.GET("", transactionHandler.GetTransactions)
			transactions.POST("", transactionHandler.CreateTransaction)
			transactions.GET("/:id", transactionHandler.GetTransactionByID)
			transactions.PUT("/:id", transactionHandler.UpdateTransaction)
			transactions.DELETE("/:id", transactionHandler.DeleteTransaction)
			transactions.POST("/:id/prepay", transactionHandler.PrepayTransaction)
			transactions.PATCH("/:id/end", transactionHandler.EndRecurringTransaction)
			transactions.POST("/by-categories", transactionHandler.GetTransactionsByCategories)
			transactions.POST("/by-category/:id", transactionHandler.GetTransactionsByCategory)
			transactions.POST("/by-date-range", transactionHandler.GetTransactionsByDateRange)
			transactions.GET("/latest", transactionHandler.GetLatestTransactions)
			transactions.GET("/biggest", transactionHandler.GetBiggestTransactions)
			transactions.GET("/average/by-type", transactionHandler.GetAverageByType)
			transactions.GET("/average/by-category", transactionHandler.GetAverageByCategory)
		}

		categories := v1.Group("/categories")
		{
			categories.GET("", categoryHandler.GetCategories)
			categories.POST("", categoryHandler.CreateCategory)
			categories.GET("/:id", categoryHandler.GetCategoryByID)
			categories.PUT("/:id", categoryHandler.UpdateCategory)
			categories.DELETE("/:id", categoryHandler.DeleteCategory)
		}

		subcategories := v1.Group("/subcategories")
		{
			subcategories.GET("", subcategoryHandler.GetSubcategories)
			subcategories.POST("", subcategoryHandler.CreateSubcategory)
			subcategories.GET("/:id", subcategoryHandler.GetSubcategoryByID)
			subcategories.PUT("/:id", subcategoryHandler.UpdateSubcategory)
			subcategories.DELETE("/:id", subcategoryHandler.DeleteSubcategory)
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
