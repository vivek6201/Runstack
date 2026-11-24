package response

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Response is the unified generic API envelope returned for all JSON endpoints.
// Every response contains request_id, success flag, and optionally data or error.
type Response[T any] struct {
	RequestID string `json:"request_id"`
	Success   bool   `json:"success"`
	Data      T      `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Envelope is an untyped API response envelope for arbitrary payloads.
type Envelope struct {
	RequestID string      `json:"request_id"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// GetRequestID extracts or generates the request ID for the current context.
func GetRequestID(c *fiber.Ctx) string {
	if c == nil {
		return uuid.New().String()
	}

	// 1. Check locals ("requestid") populated by fiber requestid middleware
	if id, ok := c.Locals("requestid").(string); ok && id != "" {
		return id
	}

	// 2. Check X-Request-ID response header
	if id := string(c.Response().Header.Peek(fiber.HeaderXRequestID)); id != "" {
		c.Locals("requestid", id)
		return id
	}

	// 3. Check X-Request-ID request header
	if id := c.Get(fiber.HeaderXRequestID); id != "" {
		c.Response().Header.Set(fiber.HeaderXRequestID, id)
		c.Locals("requestid", id)
		return id
	}

	// 4. Generate a fresh UUID and attach to response header
	newID := uuid.New().String()
	c.Locals("requestid", newID)
	c.Response().Header.Set(fiber.HeaderXRequestID, newID)
	return newID
}

// Success returns a 200 OK standard envelope with data.
func Success(c *fiber.Ctx, data interface{}) error {
	reqID := GetRequestID(c)
	return c.Status(fiber.StatusOK).JSON(Envelope{
		RequestID: reqID,
		Success:   true,
		Data:      data,
	})
}

// Created returns a 201 Created standard envelope with data.
func Created(c *fiber.Ctx, data interface{}) error {
	reqID := GetRequestID(c)
	return c.Status(fiber.StatusCreated).JSON(Envelope{
		RequestID: reqID,
		Success:   true,
		Data:      data,
	})
}

// Accepted returns a 202 Accepted standard envelope with data.
func Accepted(c *fiber.Ctx, data interface{}) error {
	reqID := GetRequestID(c)
	return c.Status(fiber.StatusAccepted).JSON(Envelope{
		RequestID: reqID,
		Success:   true,
		Data:      data,
	})
}

// JSON returns a custom status code standard envelope with data.
func JSON(c *fiber.Ctx, status int, data interface{}) error {
	reqID := GetRequestID(c)
	return c.Status(status).JSON(Envelope{
		RequestID: reqID,
		Success:   status >= 200 && status < 300,
		Data:      data,
	})
}

// Error returns a standardized error response with the given status code and message.
func Error(c *fiber.Ctx, status int, errMessage string) error {
	reqID := GetRequestID(c)
	return c.Status(status).JSON(Envelope{
		RequestID: reqID,
		Success:   false,
		Error:     errMessage,
	})
}

// BadRequest returns 400 Bad Request.
func BadRequest(c *fiber.Ctx, errMessage string) error {
	return Error(c, fiber.StatusBadRequest, errMessage)
}

// Unauthorized returns 401 Unauthorized.
func Unauthorized(c *fiber.Ctx, errMessage string) error {
	return Error(c, fiber.StatusUnauthorized, errMessage)
}

// Forbidden returns 403 Forbidden.
func Forbidden(c *fiber.Ctx, errMessage string) error {
	return Error(c, fiber.StatusForbidden, errMessage)
}

// NotFound returns 404 Not Found.
func NotFound(c *fiber.Ctx, errMessage string) error {
	return Error(c, fiber.StatusNotFound, errMessage)
}

// TooManyRequests returns 429 Too Many Requests.
func TooManyRequests(c *fiber.Ctx, errMessage string) error {
	return Error(c, fiber.StatusTooManyRequests, errMessage)
}

// InternalServerError returns 500 Internal Server Error.
func InternalServerError(c *fiber.Ctx, errMessage string) error {
	return Error(c, fiber.StatusInternalServerError, errMessage)
}

// BadGateway returns 502 Bad Gateway.
func BadGateway(c *fiber.Ctx, errMessage string) error {
	return Error(c, fiber.StatusBadGateway, errMessage)
}

// GatewayTimeout returns 504 Gateway Timeout.
func GatewayTimeout(c *fiber.Ctx, errMessage string) error {
	return Error(c, fiber.StatusGatewayTimeout, errMessage)
}

// FiberErrorHandler is a global Fiber error handler that formats any unhandled route
// or framework error (e.g. 404 Route Not Found, 405 Method Not Allowed) into the standardized envelope.
func FiberErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}
	return Error(c, code, err.Error())
}

// Decode extracts a generic Response[T] from an io.Reader.
func Decode[T any](r io.Reader) (*Response[T], error) {
	var res Response[T]
	if err := json.NewDecoder(r).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
