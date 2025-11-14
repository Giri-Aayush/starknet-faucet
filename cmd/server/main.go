package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/Giri-Aayush/starknet-faucet/internal/api"
	"github.com/Giri-Aayush/starknet-faucet/internal/cache"
	"github.com/Giri-Aayush/starknet-faucet/internal/config"
	"github.com/Giri-Aayush/starknet-faucet/internal/pow"
	"github.com/Giri-Aayush/starknet-faucet/internal/starknet"
	"github.com/Giri-Aayush/starknet-faucet/pkg/utils"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := utils.NewLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Starknet Faucet Server",
		zap.String("network", cfg.Network),
		zap.String("port", cfg.Port),
	)

	// Initialize Redis
	logger.Info("Connecting to Redis...")
	redis, err := cache.NewRedisClient(
		cfg.RedisURL,
		cfg.MaxRequestsPerDayIP,
		cfg.MaxChallengesPerHour,
	)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redis.Close()
	logger.Info("Connected to Redis",
		zap.Int("max_requests_per_day_ip", cfg.MaxRequestsPerDayIP),
		zap.Int("max_challenges_per_hour", cfg.MaxChallengesPerHour),
	)

	// Initialize Starknet client
	logger.Info("Initializing Starknet client...")
	starknetClient, err := starknet.NewFaucetClient(
		cfg.StarknetRPCURL,
		cfg.FaucetPrivateKey,
		cfg.FaucetAddress,
		cfg.ETHTokenAddress,
		cfg.STRKTokenAddress,
	)
	if err != nil {
		logger.Fatal("Failed to create Starknet client", zap.Error(err))
	}
	logger.Info("Starknet client initialized",
		zap.String("faucet_address", cfg.FaucetAddress),
	)

	// Initialize PoW generator
	powGenerator := pow.NewGenerator(cfg.PoWDifficulty, cfg.ChallengeTTL)
	logger.Info("PoW generator initialized",
		zap.Int("difficulty", cfg.PoWDifficulty),
	)

	// Create API handler
	handler := api.NewHandler(cfg, logger, redis, starknetClient, powGenerator)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "Starknet Faucet API",
		DisableStartupMessage: false,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Setup routes
	api.SetupRoutes(app, handler)

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		logger.Info("Server starting", zap.String("addr", addr))
		if err := app.Listen(addr); err != nil {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("Server stopped")
}
