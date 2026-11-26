package config

import (
	"strings"
	"time"

	"code-runner/shared/database"
	"code-runner/shared/env"
)

// ConfigVar holds all centralized configuration values for the Dashboard API service.
type ConfigVar struct {
	Port               string
	AppEnv             string
	DBConfig           *database.Config
	RedisAddr          string
	RedisPassword      string
	JWTSecret          string
	TokenTTL           time.Duration
	EngineURL          string
	DevBypassKey       string
	CORSAllowedOrigins []string
	CookieSecure       bool
	CookieDomain       string
}

// LoadConfig reads environment variables once into a typed ConfigVar struct.
func LoadConfig() *ConfigVar {
	appEnv := strings.ToLower(env.Get("APP_ENV", "development"))
	rawOrigins := env.Get("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000,http://127.0.0.1:5173")

	origins := make([]string, 0)
	for _, o := range strings.Split(rawOrigins, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	cookieSecure := env.GetBool("COOKIE_SECURE", appEnv == "production")

	return &ConfigVar{
		Port:               env.Get("DASHBOARD_PORT", env.Get("PORT", "8080")),
		AppEnv:             appEnv,
		DBConfig:           database.NewConfigFromEnv(),
		RedisAddr:          env.Get("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      env.Get("REDIS_PASSWORD", ""),
		JWTSecret:          env.Get("JWT_SECRET", "cr_dev_jwt_super_secret_key_change_in_prod"),
		TokenTTL:           time.Duration(env.GetInt("TOKEN_TTL_HOURS", 72)) * time.Hour,
		EngineURL:          env.Get("ENGINE_URL", "http://localhost:8081"),
		DevBypassKey:       env.Get("DEV_BYPASS_KEY", "cr_dev_bypass_secret"),
		CORSAllowedOrigins: origins,
		CookieSecure:       cookieSecure,
		CookieDomain:       env.Get("COOKIE_DOMAIN", ""),
	}
}
