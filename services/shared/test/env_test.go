package test

import (
	"os"
	"testing"
	"time"

	"code-runner/shared/env"
)

func TestEnv_Get(t *testing.T) {
	os.Setenv("TEST_ENV_STR", "hello_world")
	defer os.Unsetenv("TEST_ENV_STR")

	if val := env.Get("TEST_ENV_STR", "fallback"); val != "hello_world" {
		t.Errorf("expected 'hello_world', got %s", val)
	}

	if val := env.Get("TEST_UNSET_KEY", "fallback"); val != "fallback" {
		t.Errorf("expected 'fallback', got %s", val)
	}
}

func TestEnv_GetInt(t *testing.T) {
	os.Setenv("TEST_ENV_INT", "42")
	defer os.Unsetenv("TEST_ENV_INT")

	if val := env.GetInt("TEST_ENV_INT", 0); val != 42 {
		t.Errorf("expected 42, got %d", val)
	}

	if val := env.GetInt("TEST_UNSET_INT", 99); val != 99 {
		t.Errorf("expected fallback 99, got %d", val)
	}
}

func TestEnv_GetBool(t *testing.T) {
	os.Setenv("TEST_ENV_BOOL", "true")
	defer os.Unsetenv("TEST_ENV_BOOL")

	if val := env.GetBool("TEST_ENV_BOOL", false); val != true {
		t.Errorf("expected true, got %v", val)
	}

	if val := env.GetBool("TEST_UNSET_BOOL", false); val != false {
		t.Errorf("expected false, got %v", val)
	}
}

func TestEnv_GetDuration(t *testing.T) {
	os.Setenv("TEST_ENV_DUR", "10s")
	defer os.Unsetenv("TEST_ENV_DUR")

	if val := env.GetDuration("TEST_ENV_DUR", time.Minute); val != 10*time.Second {
		t.Errorf("expected 10s, got %v", val)
	}
}
