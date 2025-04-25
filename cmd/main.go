package main

import (
	"log"
	"net/http"

	"github.com/navindunimsara2001/bet-settlement-engine-vortex/internal/handler"
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/internal/repository/memory"
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// --- Dependency Injection ---
	// Create the in-memory repository
	betRepo := memory.NewInMemoryBetRepository()

	// Create the service layer
	betService := service.NewBetService(betRepo)

	// Create the application handler (which now includes user and bet handlers)
	appHandler := handler.NewAppHandler(betService)

	// --- Fiber App Setup ---
	app := fiber.New(fiber.Config{
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			// Default error status code
			code := fiber.StatusInternalServerError
			message := "Internal Server Error"

			// Check if it's a fiber.*Error
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			} else {
				// Log non-fiber errors here, as they might be unexpected
				log.Printf("Unhandled internal error: %v - Path: %s", err, ctx.Path())
			}

            // Log the error status that will be sent to the client
            log.Printf("Responding with status %d: %s - Path: %s", code, message, ctx.Path())

			// Send JSON error response
			return ctx.Status(code).JSON(fiber.Map{
				"error": message,
			})
		},
	})

	// --- Middleware ---
	app.Use(recover.New()) // Recover from panics anywhere in the chain
	app.Use(logger.New(logger.Config{ // Basic request logging
		Format: "[${time}] ${ip}:${port} ${status} - ${method} ${path} ${latency}\n",
	}))

	// --- Register Routes ---
	// The AppHandler's RegisterRoutes method sets up all /api/v1 routes
	appHandler.RegisterRoutes(app)

	// --- Health Check Endpoint ---
	app.Get("/health", func(c *fiber.Ctx) error {
		// Add more checks here if needed (e.g., DB connection)
		return c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok"})
	})

    // --- Optional: Route Not Found Handler ---
    app.Use(func(c *fiber.Ctx) error {
        log.Printf("Route not found: %s %s", c.Method(), c.Path())
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "Route not found",
        })
    })


	// --- Start Server ---
	port := "8080"
	log.Printf("Starting Bet Settlement API server on port %s...", port)

	err := app.Listen(":" + port)
	if err != nil {
		log.Fatalf("Failed to start server on port %s: %v", port, err)
	}
}