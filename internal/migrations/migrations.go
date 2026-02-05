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
	files := []string{
		"20250911_initial_schema.sql",
		"20250912_create_categories.sql",
		"20250913_add_prepaid_from_id.sql",
	}

	for _, file := range files {
		sqlContent, err := migrationFiles.ReadFile(file)
		if err != nil {
			logger.Error("failed to read migration file", "file", file, "error", err)
			return err
		}

		if err := db.Exec(string(sqlContent)).Error; err != nil {
			logger.Error("failed to execute migration", "file", file, "error", err)
			return err
		}

		logger.Info("migration executed", "file", file)
	}

	logger.Info("all migrations executed successfully")
	return nil
}
