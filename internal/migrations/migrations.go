package migrations

import (
	"embed"
	"log/slog"
	"sort"

	"gorm.io/gorm"
)

//go:embed *.sql
var migrationFiles embed.FS

func RunMigrations(db *gorm.DB, logger *slog.Logger) error {
	entries, err := migrationFiles.ReadDir(".")
	if err != nil {
		logger.Error("failed to read migration directory", "error", err)
		return err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

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
