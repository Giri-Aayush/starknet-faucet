package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// SetupRoutes sets up all API routes
func SetupRoutes(app *fiber.App, handler *Handler) {
	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	// CORS - Allow all origins for public faucet API
	// CLI and frontend can make requests from anywhere
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",  // Public API - allow all domains
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET, POST, OPTIONS",
	}))

	// Health check
	app.Get("/health", handler.Health)

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Challenge endpoint
	v1.Post("/challenge", handler.GetChallenge)

	// Faucet endpoint
	v1.Post("/faucet", handler.RequestTokens)

	// Status endpoint
	v1.Get("/status/:address", handler.GetStatus)

	// Info endpoint
	v1.Get("/info", handler.GetInfo)

	// Quota endpoint
	v1.Get("/quota", handler.GetQuota)
}
