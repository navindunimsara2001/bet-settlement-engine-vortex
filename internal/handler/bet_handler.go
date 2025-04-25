package handler

import (
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/internal/model"
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/internal/service"
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/pkg/errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// AppHandler handles HTTP requests for both bets and users.
type AppHandler struct {
	service *service.BetService
}

// NewAppHandler creates a new AppHandler.
func NewAppHandler(s *service.BetService) *AppHandler {
	return &AppHandler{service: s}
}

// RegisterRoutes registers all API routes (bets and users).
func (h *AppHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1") 

	// Bet Routes
	bets := api.Group("/bets")
	{
		bets.Post("/", h.PlaceBet)               
		bets.Post("/settle/:eventId", h.SettleBet) 
	}

	// User Routes
	users := api.Group("/users")
	{
		users.Post("/", h.CreateUser)             
		users.Get("/", h.ListUsers)               
		users.Get("/:userId", h.GetUser)          
		users.Get("/:userId/balance", h.GetUserBalance) 
		users.Put("/:userId", h.UpdateUser)      
		users.Delete("/:userId", h.DeleteUser) 
	}
}

// --- Bet Handlers ---

// PlaceBet handles the request to place a new bet.
// @Summary Place a new bet
// @Description Places a bet for a user on a specific event.
// @Tags Bets
// @Accept json
// @Produce json
// @Param bet body model.PlaceBetRequest true "Bet details"
// @Success 201 {object} model.Bet "Bet placed successfully"
// @Failure 400 {object} map[string]string "Bad Request (validation error, insufficient balance)"
// @Failure 404 {object} map[string]string "Not Found (user creation failed)"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /bets [post]
func (h *AppHandler) PlaceBet(c *fiber.Ctx) error {
	var req model.PlaceBetRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing request body for PlaceBet: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON request body"})
	}

	if err := req.Validate(); err != nil {
		log.Printf("Validation failed for PlaceBet request: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Validation failed: %s", err.Error())})
	}

	bet, err := h.service.PlaceBet(&req)
	if err != nil {
		log.Printf("Service error in PlaceBet: %v", err)
		if e, ok := err.(*errors.ErrorBadRequest); ok {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": e.Error()})
		}
		if e, ok := err.(*errors.ErrorNotFound); ok { 
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": e.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to place bet"})
	}

	return c.Status(http.StatusCreated).JSON(bet)
}

// SettleBet handles the request to settle bets for an event.
// @Summary Settle bets for an event
// @Description Settles all 'placed' bets for a given event ID based on the result (win/lose).
// @Tags Bets
// @Accept json
// @Produce json
// @Param eventId path string true "Event ID"
// @Param result body model.SettleBetRequest true "Settlement result"
// @Success 200 {object} map[string]string "Bets settled successfully"
// @Failure 400 {object} map[string]string "Bad Request (invalid event ID or result)"
// @Failure 404 {object} map[string]string "Not Found (no placed bets for the event)"
// @Failure 409 {object} map[string]string "Conflict (e.g., bet already settled)"
// @Failure 500 {object} map[string]string "Internal Server Error (partial settlement possible)"
// @Router /bets/settle/{eventId} [post]
func (h *AppHandler) SettleBet(c *fiber.Ctx) error {
	eventID := c.Params("eventId")
	if eventID == "" {
        log.Print("SettleBet request missing eventId")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Event ID is required"})
	}

	var req model.SettleBetRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing request body for SettleBet (event: %s): %v", eventID, err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON request body"})
	}

	// Basic validation
	if err := req.Validate(); err != nil {
		log.Printf("Validation failed for SettleBet request (event: %s): %v", eventID, err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("Validation failed: %s", err.Error())})
	}

	err := h.service.SettleBetsForEvent(eventID, req.Result)
	if err != nil {
		log.Printf("Service error in SettleBet (event: %s): %v", eventID, err)
		if e, ok := err.(*errors.ErrorNotFound); ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": e.Error()})
		}
		if e, ok := err.(*errors.ErrorBadRequest); ok {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": e.Error()})
		}
		if e, ok := err.(*errors.ErrorConflict); ok { 
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": e.Error(), "message": "Some bets might have been settled previously or failed."})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("Failed to settle all bets for event %s, potential partial success: %s", eventID, err.Error())})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": fmt.Sprintf("Bets for event %s settled successfully with result '%s'", eventID, req.Result)})
}

// --- User CRUD Handlers ---

// CreateUser handles the request to create a new user.
// @Summary Create a new user
// @Description Creates a user with a given ID and default balance.
// @Tags Users
// @Accept json
// @Produce json
// @Param user body model.CreateUserRequest true "User details"
// @Success 201 {object} model.User "User created successfully"
// @Failure 400 {object} map[string]string "Bad Request (validation error)"
// @Failure 409 {object} map[string]string "Conflict (user already exists)"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /users [post]
func (h *AppHandler) CreateUser(c *fiber.Ctx) error {
	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing request body for CreateUser: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON request body"})
	}

	// Service layer handles validation
	user, err := h.service.CreateUser(&req)
	if err != nil {
		log.Printf("Service error in CreateUser: %v", err)
		if e, ok := err.(*errors.ErrorBadRequest); ok {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": e.Error()})
		}
		if e, ok := err.(*errors.ErrorConflict); ok {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": e.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create user"})
	}

	return c.Status(http.StatusCreated).JSON(user)
}

// GetUser handles the request to retrieve a user by ID.
// @Summary Get user by ID
// @Description Retrieves details for a specific user ID.
// @Tags Users
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} model.User "User details"
// @Failure 400 {object} map[string]string "Bad Request (invalid user ID)"
// @Failure 404 {object} map[string]string "Not Found (user does not exist)"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /users/{userId} [get]
func (h *AppHandler) GetUser(c *fiber.Ctx) error {
	userID := c.Params("userId")

	user, err := h.service.GetUser(userID)
	if err != nil {
		log.Printf("Service error in GetUser (user: %s): %v", userID, err)
		if e, ok := err.(*errors.ErrorNotFound); ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": e.Error()})
		}
		if e, ok := err.(*errors.ErrorBadRequest); ok {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": e.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve user"})
	}

	return c.Status(http.StatusOK).JSON(user)
}

// ListUsers handles the request to retrieve all users.
// @Summary List all users
// @Description Retrieves a list of all registered users.
// @Tags Users
// @Produce json
// @Success 200 {array} model.User "List of users"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /users [get]
func (h *AppHandler) ListUsers(c *fiber.Ctx) error {
	users, err := h.service.ListUsers()
	if err != nil {
		log.Printf("Service error in ListUsers: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve users"})
	}
	return c.Status(http.StatusOK).JSON(users)
}

// UpdateUser handles the request to update a user.
// @Summary Update a user
// @Description Updates details for a specific user ID. (Note: Current implementation might only update timestamps).
// @Tags Users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param user body model.UpdateUserRequest true "User details to update"
// @Success 200 {object} model.User "User updated successfully"
// @Failure 400 {object} map[string]string "Bad Request (invalid user ID or validation error)"
// @Failure 404 {object} map[string]string "Not Found (user does not exist)"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /users/{userId} [put]
func (h *AppHandler) UpdateUser(c *fiber.Ctx) error {
	userID := c.Params("userId")
	var req model.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing request body for UpdateUser (user: %s): %v", userID, err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON request body"})
	}

	// Service layer handles validation and finding the user
	user, err := h.service.UpdateUser(userID, &req)
	if err != nil {
		log.Printf("Service error in UpdateUser (user: %s): %v", userID, err)
		if e, ok := err.(*errors.ErrorNotFound); ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": e.Error()})
		}
		if e, ok := err.(*errors.ErrorBadRequest); ok {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": e.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update user"})
	}

	return c.Status(http.StatusOK).JSON(user)
}

// DeleteUser handles the request to delete a user.
// @Summary Delete a user
// @Description Deletes a specific user ID.
// @Tags Users
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]string "User deleted successfully"
// @Failure 400 {object} map[string]string "Bad Request (invalid user ID)"
// @Failure 404 {object} map[string]string "Not Found (user does not exist)"
// @Failure 409 {object} map[string]string "Conflict (e.g., user has active bets, if check implemented)"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /users/{userId} [delete]
func (h *AppHandler) DeleteUser(c *fiber.Ctx) error {
	userID := c.Params("userId")

	err := h.service.DeleteUser(userID)
	if err != nil {
		log.Printf("Service error in DeleteUser (user: %s): %v", userID, err)
		if e, ok := err.(*errors.ErrorNotFound); ok {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": e.Error()})
		}
		if e, ok := err.(*errors.ErrorBadRequest); ok {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": e.Error()})
		}
		if e, ok := err.(*errors.ErrorConflict); ok { 
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": e.Error()})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete user"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"message": fmt.Sprintf("User %s deleted successfully", userID)})
}


// GetUserBalance handles the request to get a user's balance.
// @Summary Get user balance
// @Description Retrieves the current balance for a specific user ID.
// @Tags Users
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]float64 "User balance"
// @Failure 400 {object} map[string]string "Bad Request (invalid user ID)"
// @Failure 404 {object} map[string]string "Not Found (user does not exist)"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /users/{userId}/balance [get]
func (h *AppHandler) GetUserBalance(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
        log.Print("GetUserBalance request missing userId")
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "User ID is required"})
	}

	balance, err := h.service.GetUserBalance(userID)
	if err != nil {
		log.Printf("Service error in GetUserBalance (user: %s): %v", userID, err)
		if e, ok := err.(*errors.ErrorNotFound); ok {
			// If user doesn't exist, return 404 as per CRUD operations standard
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": e.Error()})
		}
        if e, ok := err.(*errors.ErrorBadRequest); ok {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": e.Error()})
		}
		// Other potential errors
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve user balance"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"user_id": userID, "balance": balance})
}