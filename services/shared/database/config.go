package database

import (
	"fmt"
	"strings"
	"time"

	"code-runner/shared/env"
	"code-runner/shared/logger"
)

// Config holds all database connection parameters and pool settings.
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	LogLevel        string
}

// NewConfigFromEnv parses database settings from environment variables.
// It prioritizes DATABASE_URL if present, otherwise constructs a DSN from discrete variables.
func NewConfigFromEnv() *Config {
	appEnv := strings.ToLower(env.Get("APP_ENV", "development"))
	dsn := env.Get("DATABASE_URL", "")
	if dsn == "" {
		host := env.Get("DB_HOST", "localhost")
		port := env.GetInt("DB_PORT", 5432)
		user := env.Get("DB_USER", "postgres")
		pass := env.Get("DB_PASSWORD", "postgres")
		name := env.Get("DB_NAME", "coderunner")

		defaultSSL := "disable"
		if appEnv == "production" {
			defaultSSL = "require"
		}
		ssl := env.Get("DB_SSLMODE", defaultSSL)
		if appEnv == "production" && ssl == "disable" {
			logger.Info("SECURITY WARNING: DB_SSLMODE=disable is unsafe in production environments")
		}

		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			user, pass, host, port, name, ssl,
		)
	} else if appEnv == "production" && strings.Contains(strings.ToLower(dsn), "sslmode=disable") {
		logger.Info("SECURITY WARNING: DATABASE_URL uses sslmode=disable in production environments")
	}

	return &Config{
		DSN:             dsn,
		MaxOpenConns:    env.GetInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    env.GetInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: env.GetDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		ConnMaxIdleTime: env.GetDuration("DB_CONN_MAX_IDLE_TIME", 1*time.Minute),
		LogLevel:        env.Get("DB_LOG_LEVEL", "warn"),
	}
}
