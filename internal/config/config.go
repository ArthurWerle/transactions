package config

import (
	"fmt"
	"os"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Log       LogConfig
	Reporting ReportingConfig
	Identity  IdentityConfig
}

type IdentityConfig struct {
	// BaseURL is the identity service address reachable on the internal
	// docker network. Used to resolve the display name of a transaction's
	// creator. Empty disables enrichment.
	BaseURL string
}

type ReportingConfig struct {
	// Timezone defines the calendar used for all month/day bucketing in
	// reports (e.g. which month a late-night transaction belongs to).
	Timezone string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type LogConfig struct {
	Level string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "transactions"),
			Password: getEnv("DB_PASSWORD", "transactions_dev_password"),
			Name:     getEnv("DB_NAME", "transactions_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Reporting: ReportingConfig{
			Timezone: getEnv("REPORTING_TIMEZONE", "America/Sao_Paulo"),
		},
		Identity: IdentityConfig{
			BaseURL: getEnv("IDENTITY_BASE_URL", "http://identity:8080"),
		},
	}

	return cfg, nil
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
