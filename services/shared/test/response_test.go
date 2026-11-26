package test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"code-runner/shared/response"
)

func TestResponse_Success(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c fiber.Ctx) error {
		return response.Success(c, fiber.Map{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var env response.Response[map[string]string]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode response envelope: %v", err)
	}

	if !env.Success {
		t.Errorf("expected success to be true")
	}
	if env.RequestID == "" {
		t.Errorf("expected request_id to be populated")
	}
	if env.Data["message"] != "ok" {
		t.Errorf("expected data.message to be 'ok', got %q", env.Data["message"])
	}
	if env.Error != "" {
		t.Errorf("expected error to be empty on success, got %q", env.Error)
	}
}

func TestResponse_Error(t *testing.T) {
	app := fiber.New()
	app.Get("/error", func(c fiber.Ctx) error {
		return response.NotFound(c, "resource not found")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	var env response.Response[interface{}]
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode response envelope: %v", err)
	}

	if env.Success {
		t.Errorf("expected success to be false")
	}
	if env.RequestID == "" {
		t.Errorf("expected request_id to be populated")
	}
	if env.Error != "resource not found" {
		t.Errorf("expected error message 'resource not found', got %q", env.Error)
	}
}

func TestResponse_FiberErrorHandler(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: response.FiberErrorHandler,
	})
	app.Get("/panic", func(c fiber.Ctx) error {
		return errors.New("something went wrong")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	var env response.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode response envelope: %v", err)
	}

	if env.Success {
		t.Errorf("expected success to be false")
	}
	if env.Error != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got %q", env.Error)
	}
}
