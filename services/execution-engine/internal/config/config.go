package config

import (
	"strings"

	"code-runner/shared/env"
)

// ConfigVar holds centralized configuration for the Execution Engine service.
type ConfigVar struct {
	Port            string
	AppEnv          string
	RedisAddr       string
	RedisPassword   string
	DevBypassKey    string
	NumWorkers      int
	ExecutionDriver string
}

// LoadConfig loads environment variables once into ConfigVar.
func LoadConfig() *ConfigVar {
	return &ConfigVar{
		Port:            env.Get("ENGINE_PORT", env.Get("PORT", "8081")),
		AppEnv:          strings.ToLower(env.Get("APP_ENV", "development")),
		RedisAddr:       env.Get("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   env.Get("REDIS_PASSWORD", ""),
		DevBypassKey:    env.Get("DEV_BYPASS_KEY", "cr_dev_bypass_secret"),
		NumWorkers:      env.GetInt("NUM_WORKERS", 4),
		ExecutionDriver: strings.ToLower(env.Get("EXECUTION_DRIVER", "local")),
	}
}
