package migrations

import (
	"embed"
	"log/slog"

	"gorm.io/gorm"
)

//go:embed *.sql
var migrationFiles embed.FS

// RunMigrations executes all SQL migration files
func RunMigrations(db *gorm.DB, logger *slog.Logger) error {
	// Read and execute the initial schema migration
	sqlContent, err := migrationFiles.ReadFile("20250911_initial_schema.sql")
	if err != nil {
		logger.Error("failed to read migration file", "error", err)
		return err
	}

	if err := db.Exec(string(sqlContent)).Error; err != nil {
		logger.Error("failed to execute migration", "error", err)
		return err
	}

	logger.Info("migrations executed successfully")
	return nil
}
