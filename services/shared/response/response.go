package response

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/google/uuid"
)

// Response is the unified generic API envelope returned for all JSON endpoints.
type Response[T any] struct {
	RequestID string `json:"request_id"`
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Data      T      `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

// Envelope is an untyped API response envelope for arbitrary payloads.
type Envelope struct {
	RequestID string      `json:"request_id"`
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// GetRequestID extracts or generates the request ID for the current Fiber v3 context.
func GetRequestID(c fiber.Ctx) string {
	if c == nil {
		return uuid.New().String()
	}

	if id := requestid.FromContext(c); id != "" {
		return id
	}

	if id, ok := c.Locals("requestid").(string); ok && id != "" {
		return id
	}

	if id := c.GetRespHeader(fiber.HeaderXRequestID); id != "" {
		c.Locals("requestid", id)
		return id
	}

	if id := c.Get(fiber.HeaderXRequestID); id != "" {
		c.Set(fiber.HeaderXRequestID, id)
		c.Locals("requestid", id)
		return id
	}

	newID := uuid.New().String()
	c.Locals("requestid", newID)
	c.Set(fiber.HeaderXRequestID, newID)
	return newID
}

// SendSuccess writes a standardized JSON success envelope (biolynq-style).
func SendSuccess(c fiber.Ctx, status int, message string, data interface{}) error {
	return c.Status(status).JSON(Envelope{
		RequestID: GetRequestID(c),
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// SendError writes a standardized JSON error envelope (biolynq-style).
func SendError(c fiber.Ctx, status int, errMessage string) error {
	return c.Status(status).JSON(Envelope{
		RequestID: GetRequestID(c),
		Success:   false,
		Message:   errMessage,
		Error:     errMessage,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Success returns a 200 OK standard envelope with data.
func Success(c fiber.Ctx, data interface{}) error {
	return SendSuccess(c, fiber.StatusOK, "", data)
}

// Created returns a 201 Created standard envelope with data.
func Created(c fiber.Ctx, data interface{}) error {
	return SendSuccess(c, fiber.StatusCreated, "", data)
}

// Accepted returns a 202 Accepted standard envelope with data.
func Accepted(c fiber.Ctx, data interface{}) error {
	return SendSuccess(c, fiber.StatusAccepted, "", data)
}

// JSON returns a custom status code standard envelope with data.
func JSON(c fiber.Ctx, status int, data interface{}) error {
	if status >= 200 && status < 300 {
		return SendSuccess(c, status, "", data)
	}
	return c.Status(status).JSON(Envelope{
		RequestID: GetRequestID(c),
		Success:   false,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Error returns a standardized error response with the given status code and message.
func Error(c fiber.Ctx, status int, errMessage string) error {
	return SendError(c, status, errMessage)
}

// BadRequest returns 400 Bad Request.
func BadRequest(c fiber.Ctx, errMessage string) error {
	return SendError(c, fiber.StatusBadRequest, errMessage)
}

// Unauthorized returns 401 Unauthorized.
func Unauthorized(c fiber.Ctx, errMessage string) error {
	return SendError(c, fiber.StatusUnauthorized, errMessage)
}

// Forbidden returns 403 Forbidden.
func Forbidden(c fiber.Ctx, errMessage string) error {
	return SendError(c, fiber.StatusForbidden, errMessage)
}

// NotFound returns 404 Not Found.
func NotFound(c fiber.Ctx, errMessage string) error {
	return SendError(c, fiber.StatusNotFound, errMessage)
}

// TooManyRequests returns 429 Too Many Requests.
func TooManyRequests(c fiber.Ctx, errMessage string) error {
	return SendError(c, fiber.StatusTooManyRequests, errMessage)
}

// InternalServerError returns 500 Internal Server Error.
func InternalServerError(c fiber.Ctx, errMessage string) error {
	return SendError(c, fiber.StatusInternalServerError, errMessage)
}

// BadGateway returns 502 Bad Gateway.
func BadGateway(c fiber.Ctx, errMessage string) error {
	return SendError(c, fiber.StatusBadGateway, errMessage)
}

// GatewayTimeout returns 504 Gateway Timeout.
func GatewayTimeout(c fiber.Ctx, errMessage string) error {
	return SendError(c, fiber.StatusGatewayTimeout, errMessage)
}

// FiberErrorHandler formats any unhandled route or validation error into the standardized envelope.
func FiberErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}
	return SendError(c, code, err.Error())
}

// Decode extracts a generic Response[T] from an io.Reader.
func Decode[T any](r io.Reader) (*Response[T], error) {
	var res Response[T]
	if err := json.NewDecoder(r).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
