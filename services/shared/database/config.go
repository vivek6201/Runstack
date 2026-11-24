package database

import (
	"fmt"
	"time"

	"code-runner/shared/env"
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
	dsn := env.Get("DATABASE_URL", "")
	if dsn == "" {
		host := env.Get("DB_HOST", "localhost")
		port := env.GetInt("DB_PORT", 5432)
		user := env.Get("DB_USER", "postgres")
		pass := env.Get("DB_PASSWORD", "postgres")
		name := env.Get("DB_NAME", "coderunner")
		ssl := env.Get("DB_SSLMODE", "disable")

		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			user, pass, host, port, name, ssl,
		)
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
