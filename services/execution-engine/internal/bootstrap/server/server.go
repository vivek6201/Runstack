package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	fiberLogger "github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"code-runner/execution-engine/internal/adapters/executor"
	"code-runner/execution-engine/internal/adapters/executor/firecracker"
	"code-runner/execution-engine/internal/adapters/queue"
	"code-runner/execution-engine/internal/config"
	"code-runner/execution-engine/internal/ports"
	"code-runner/execution-engine/internal/worker"
	"code-runner/shared/database"
	"code-runner/shared/logger"
	"code-runner/shared/response"
	sharedValidator "code-runner/shared/validator"
)

// StartServer initializes Redis, worker pools, executor adapters, and the Fiber v3 HTTP server.
func StartServer(cfg *config.ConfigVar) {
	logger.Info(fmt.Sprintf("Starting Execution Engine on port :%s", cfg.Port))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisClient, err := database.ConnectRedis(cfg.RedisAddr, cfg.RedisPassword)
	queueAdapter := queue.NewRedisQueueAdapter(redisClient)

	if err != nil {
		logger.Info(fmt.Sprintf("Warning: Redis not reachable at %s (%v). Start Redis via 'make infra-up'", cfg.RedisAddr, err))
	} else {
		logger.Info(fmt.Sprintf("Connected to Redis at %s", cfg.RedisAddr))
		if requeued, recErr := queueAdapter.RequeueStale(ctx, 2*time.Minute); recErr == nil && requeued > 0 {
			logger.Info(fmt.Sprintf("Recovered %d orphaned job(s) back to pending queue", requeued))
		}
	}

	var execAdapter ports.Executor
	if cfg.ExecutionDriver == "firecracker" {
		fcAdapter, fcErr := firecracker.NewFirecrackerAdapter(ctx, nil)
		if fcErr != nil {
			logger.Info(fmt.Sprintf("Notice: cannot start Firecracker (%v). Falling back to local runner.", fcErr))
			execAdapter = executor.NewLocalSubprocessAdapter()
		} else {
			execAdapter = fcAdapter
		}
	} else {
		execAdapter = executor.NewLocalSubprocessAdapter()
	}

	workerPool := worker.NewPool(cfg.NumWorkers, queueAdapter, queueAdapter, execAdapter)
	workerPool.Start(ctx)

	app := fiber.New(fiber.Config{
		AppName:         "Runstack Execution Engine",
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

	SetupRoutes(app, redisClient, queueAdapter, cfg)

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

	if err := app.Listen(":" + cfg.Port); err != nil {
		logger.Info(fmt.Sprintf("Execution Engine stopped: %v", err))
	}
}
