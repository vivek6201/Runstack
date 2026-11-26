package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	fiberLogger "github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"gorm.io/gorm"

	"code-runner/dashboard/internal/config"
	"code-runner/shared/database"
	"code-runner/shared/logger"
	"code-runner/shared/response"
	sharedValidator "code-runner/shared/validator"
)

// StartServer initializes Postgres, Redis, Fiber v3 HTTP server, middleware, and graceful shutdown.
func StartServer(cfg *config.ConfigVar) {
	logger.Info(fmt.Sprintf("Starting Dashboard API service on port :%s", cfg.Port))

	var db *gorm.DB
	dbConn, err := database.Connect(cfg.DBConfig)
	if err != nil {
		logger.Info(fmt.Sprintf("Warning: PostgreSQL not reachable (%v). Run 'make infra-up' to start database.", err))
	} else {
		db = dbConn
		logger.Info("Database connected successfully (schema managed via Atlas migrations).")
	}

	rdb, err := database.ConnectRedis(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		logger.Info(fmt.Sprintf("Warning: Redis not reachable at %s (%v).", cfg.RedisAddr, err))
	} else {
		logger.Info(fmt.Sprintf("Connected to Redis at %s", cfg.RedisAddr))
	}

	app := fiber.New(fiber.Config{
		AppName:         "Runstack Dashboard API",
		CaseSensitive:   true,
		StructValidator: sharedValidator.NewStructValidator(),
		ErrorHandler:    response.FiberErrorHandler,
		TrustProxy:      true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback:  true,
			Private:   true,
			LinkLocal: true,
		},
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(fiberLogger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowCredentials: true,
	}))

	SetupRoutes(app, db, rdb, cfg)

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		logger.Info("Gracefully shutting down Dashboard API...")
		_ = app.Shutdown()
	}()

	if err := app.Listen(":" + cfg.Port); err != nil {
		logger.Info(fmt.Sprintf("Server stopped: %v", err))
	}
}
