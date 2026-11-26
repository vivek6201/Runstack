package test

import (
	"strings"
	"testing"

	"code-runner/shared/database"
)

func TestDatabaseConfig_ProductionSSLDefault(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_SSLMODE", "")

	t.Setenv("APP_ENV", "production")
	prodCfg := database.NewConfigFromEnv()
	if !strings.Contains(prodCfg.DSN, "sslmode=require") {
		t.Errorf("expected sslmode=require in production default DSN, got: %s", prodCfg.DSN)
	}

	t.Setenv("APP_ENV", "development")
	devCfg := database.NewConfigFromEnv()
	if !strings.Contains(devCfg.DSN, "sslmode=disable") {
		t.Errorf("expected sslmode=disable in development default DSN, got: %s", devCfg.DSN)
	}
}
