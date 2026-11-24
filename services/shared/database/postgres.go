package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"code-runner/shared/models"
)

// Connect initializes a PostgreSQL connection via GORM and configures connection pooling.
func Connect(cfg *Config) (*gorm.DB, error) {
	if cfg == nil {
		cfg = NewConfigFromEnv()
	}

	logLevel := gormLogger.Warn
	switch cfg.LogLevel {
	case "silent":
		logLevel = gormLogger.Silent
	case "error":
		logLevel = gormLogger.Error
	case "info":
		logLevel = gormLogger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: gormLogger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Apply connection pool settings
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return db, nil
}

// AutoMigrate migrates all shared GORM entity schemas into the connected database.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.APIKey{},
		&models.Submission{},
	)
}
