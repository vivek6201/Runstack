package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Get returns the string value of the environment variable or fallback if unset/empty.
func Get(key string, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

// GetInt returns the integer value of the environment variable or fallback if invalid/unset.
func GetInt(key string, fallback int) int {
	valStr := Get(key, "")
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return val
}

// GetBool returns the boolean value of the environment variable (e.g. "true", "1", "yes") or fallback.
func GetBool(key string, fallback bool) bool {
	valStr := strings.ToLower(Get(key, ""))
	if valStr == "" {
		return fallback
	}
	return valStr == "true" || valStr == "1" || valStr == "yes" || valStr == "on"
}

// GetDuration returns the parsed time.Duration (e.g. "5m", "10s", "1h") or fallback.
func GetDuration(key string, fallback time.Duration) time.Duration {
	valStr := Get(key, "")
	if valStr == "" {
		return fallback
	}
	d, err := time.ParseDuration(valStr)
	if err != nil {
		return fallback
	}
	return d
}

// Require returns the string value of the environment variable or panics if unset/empty.
func Require(key string) string {
	val := Get(key, "")
	if val == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return val
}
