package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/redis/go-redis/v9"

	"code-runner/execution-engine/internal/adapters/auth"
	"code-runner/execution-engine/internal/adapters/executor"
	"code-runner/execution-engine/internal/adapters/executor/firecracker"
	"code-runner/execution-engine/internal/adapters/queue"
	"code-runner/execution-engine/internal/api"
	"code-runner/execution-engine/internal/ports"
	"code-runner/execution-engine/internal/routes"
	"code-runner/execution-engine/internal/worker"
	"code-runner/shared/cache"
	"code-runner/shared/env"
	"code-runner/shared/logger"
	"code-runner/shared/response"
)

func main() {
	port := env.Get("ENGINE_PORT", env.Get("PORT", "8081"))
	redisAddr := env.Get("REDIS_ADDR", "localhost:6379")
	redisPass := env.Get("REDIS_PASSWORD", "")
	devBypassKey := env.Get("DEV_BYPASS_KEY", "cr_dev_bypass_secret")
	numWorkers := env.GetInt("NUM_WORKERS", 4)

	logger.Info(fmt.Sprintf("Starting Execution Engine on port :%s", port))

	// 1. Initialize Redis Client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       0,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ping Redis in background
	go func() {
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer pingCancel()
		if err := redisClient.Ping(pingCtx).Err(); err != nil {
			logger.Info(fmt.Sprintf("Warning: Redis not reachable at %s (%v). Start Redis via 'bun run infra:up'", redisAddr, err))
		} else {
			logger.Info(fmt.Sprintf("Connected to Redis at %s", redisAddr))
		}
	}()

	// 2. Initialize Adapters (Adapter Pattern)
	queueAdapter := queue.NewRedisQueueAdapter(redisClient)
	authValidator := auth.NewRedisKeyValidator(redisClient)

	// Select Execution Driver: "firecracker" (Linux KVM) or "local" (macOS/subprocess)
	driverType := env.Get("EXECUTION_DRIVER", "local")
	var execAdapter ports.Executor

	if driverType == "firecracker" {
		fcAdapter, err := firecracker.NewFirecrackerAdapter(ctx, nil)
		if err != nil {
			logger.Info(fmt.Sprintf("Notice: cannot start Firecracker (%v). Falling back to local runner.", err))
			execAdapter = executor.NewLocalSubprocessAdapter()
		} else {
			execAdapter = fcAdapter
		}
	} else {
		execAdapter = executor.NewLocalSubprocessAdapter()
	}

	// 3. Start Background Worker Pool
	workerPool := worker.NewPool(numWorkers, queueAdapter, queueAdapter, execAdapter)
	workerPool.Start(ctx)

	// 4. Initialize Fiber HTTP Server
	app := fiber.New(fiber.Config{
		AppName:               "Code-Runner Execution Engine",
		DisableStartupMessage: false,
		ErrorHandler:          response.FiberErrorHandler,
	})

	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(fiberLogger.New())

	// 5. Register Modular Routes
	redisCache := cache.New(redisClient)
	handler := api.NewHandler(queueAdapter, queueAdapter)
	routes.RegisterRoutes(app, routes.RouterConfig{
		Handler:       handler,
		AuthValidator: authValidator,
		DevBypassKey:  devBypassKey,
		Cache:         redisCache,
	})

	// 6. Graceful Shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		logger.Info("Gracefully shutting down Execution Engine...")
		cancel()
		workerPool.Stop()
		if closer, ok := execAdapter.(interface{ Close() }); ok {
			closer.Close()
		}
		_ = app.Shutdown()
	}()

	if err := app.Listen(":" + port); err != nil {
		logger.Info(fmt.Sprintf("Execution Engine stopped: %v", err))
	}
}
