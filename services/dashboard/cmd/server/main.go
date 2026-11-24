package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"code-runner/dashboard/internal/auth"
	"code-runner/dashboard/internal/keys"
	"code-runner/dashboard/internal/playground"
	"code-runner/dashboard/internal/routes"
	"code-runner/dashboard/internal/stats"
	"code-runner/dashboard/internal/submissions"
	apiKeyRepo "code-runner/dashboard/internal/repository/apikey"
	subRepo "code-runner/dashboard/internal/repository/submission"
	userRepo "code-runner/dashboard/internal/repository/user"
	"code-runner/shared/cache"
	"code-runner/shared/database"
	"code-runner/shared/env"
	"code-runner/shared/logger"
	"code-runner/shared/response"
)

func main() {
	port := env.Get("DASHBOARD_PORT", env.Get("PORT", "8080"))
	jwtSecret := env.Get("JWT_SECRET", "cr_dev_jwt_super_secret_key_change_in_prod")
	tokenTTLHours := env.GetInt("TOKEN_TTL_HOURS", 72)
	engineURL := env.Get("ENGINE_URL", "http://localhost:8081")
	devBypassKey := env.Get("DEV_BYPASS_KEY", "cr_dev_bypass_secret")
	redisAddr := env.Get("REDIS_ADDR", "localhost:6379")
	redisPass := env.Get("REDIS_PASSWORD", "")

	logger.Info(fmt.Sprintf("Starting Dashboard API service on port :%s", port))

	// 1. Initialize PostgreSQL Connection & Migrations
	var db *gorm.DB
	dbConn, err := database.Connect(nil)
	if err != nil {
		logger.Info(fmt.Sprintf("Warning: PostgreSQL not reachable (%v). Run 'bun run infra:up' to start database.", err))
	} else {
		db = dbConn
		if err := database.AutoMigrate(db); err != nil {
			logger.Info(fmt.Sprintf("Warning: AutoMigrate failed: %v", err))
		} else {
			logger.Info("Database connected and schema auto-migrated successfully.")
		}
	}

	// 2. Initialize Redis Connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       0,
	})

	go func() {
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer pingCancel()
		if err := redisClient.Ping(pingCtx).Err(); err != nil {
			logger.Info(fmt.Sprintf("Warning: Redis not reachable at %s (%v).", redisAddr, err))
		} else {
			logger.Info(fmt.Sprintf("Connected to Redis at %s", redisAddr))
		}
	}()

	// 3. Initialize Reusable Repositories & Cache Helper
	userRepository := userRepo.NewRepository(db)
	keyRepository := apiKeyRepo.NewRepository(db)
	submissionRepository := subRepo.NewRepository(db)
	redisCache := cache.New(redisClient)

	// 4. Initialize Services and Handlers with Unified Caching
	authService := auth.NewService(userRepository, jwtSecret, time.Duration(tokenTTLHours)*time.Hour)
	authHandler := auth.NewHandler(authService)

	keysService := keys.NewService(keyRepository, redisCache)
	keysHandler := keys.NewHandler(keysService)

	subService := submissions.NewService(submissionRepository, redisCache)
	subHandler := submissions.NewHandler(subService)

	statsService := stats.NewService(submissionRepository, redisCache)
	statsHandler := stats.NewHandler(statsService)

	playHandler := playground.NewHandler(submissionRepository, redisCache, engineURL, devBypassKey)

	// 5. Initialize Fiber HTTP Server
	app := fiber.New(fiber.Config{
		AppName:               "Code-Runner Dashboard API",
		DisableStartupMessage: false,
		ErrorHandler:          response.FiberErrorHandler,
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(fiberLogger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173,http://localhost:3000,http://127.0.0.1:5173",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-API-Key",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// 6. Register Modular Routes
	routes.RegisterRoutes(app, routes.RouterConfig{
		AuthHandler:        authHandler,
		KeysHandler:        keysHandler,
		SubmissionsHandler: subHandler,
		StatsHandler:       statsHandler,
		PlaygroundHandler:  playHandler,
		JWTSecret:          jwtSecret,
	})

	// 7. Graceful Shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		logger.Info("Gracefully shutting down Dashboard API...")
		_ = app.Shutdown()
	}()

	if err := app.Listen(":" + port); err != nil {
		logger.Info(fmt.Sprintf("Server stopped: %v", err))
	}
}
