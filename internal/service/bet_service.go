package service

import (
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/internal/model"
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/internal/repository/memory"
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/pkg/errors"
	"fmt"
	"log" // Added for logging [cite: 3]
)

// BetService handles the business logic for bets.
type BetService struct {
	repo *memory.InMemoryBetRepository
}

// NewBetService creates a new BetService.
func NewBetService(repo *memory.InMemoryBetRepository) *BetService {
	return &BetService{repo: repo}
}

func (s *BetService) PlaceBet(req *model.PlaceBetRequest) (*model.Bet, error) {
	// Validate input
	if err := req.Validate(); err != nil {
		log.Printf("Validation error placing bet for user %s: %v", req.UserID, err) // Logging [cite: 3]
		return nil, &errors.ErrorBadRequest{Message: fmt.Sprintf("validation failed: %s", err.Error())}
	}

	// Ensure user exists (or create)
	_, err := s.repo.FindOrCreateUser(req.UserID)
	if err != nil {
        log.Printf("Error finding/creating user %s: %v", req.UserID, err)
		return nil, fmt.Errorf("could not ensure user exists: %w", err)
	}


	bet := &model.Bet{
		UserID:  req.UserID,
		EventID: req.EventID,
		Odds:    req.Odds,
		Amount:  req.Amount,
	}

	createdBet, err := s.repo.PlaceBet(bet)
	if err != nil {
		log.Printf("Error placing bet in repository for user %s: %v", req.UserID, err) 
		if _, ok := err.(*errors.ErrorBadRequest); ok {
            return nil, err
        }
		if _, ok := err.(*errors.ErrorNotFound); ok {
            return nil, err
        }
		return nil, fmt.Errorf("failed to place bet: %w", err)
	}
	log.Printf("Bet placed successfully: ID=%s, UserID=%s, EventID=%s", createdBet.ID, createdBet.UserID, createdBet.EventID) 
	return createdBet, nil
}


func (s *BetService) SettleBetsForEvent(eventID string, result string) error {
	
	if result != "win" && result != "lose" {
        errMsg := fmt.Sprintf("invalid settlement result '%s', must be 'win' or 'lose'", result)
        log.Printf("Error settling event %s: %s", eventID, errMsg) 
		return &errors.ErrorBadRequest{Message: errMsg}
	}

	betsToSettle, err := s.repo.FindBetsByEvent(eventID)
	if err != nil {
		if _, ok := err.(*errors.ErrorNotFound); ok {
            log.Printf("No placed bets found to settle for event %s", eventID)
			return err 
		}
        log.Printf("Error finding bets for event %s: %v", eventID, err) 
		return fmt.Errorf("failed to find bets for event %s: %w", eventID, err)
	}

    if len(betsToSettle) == 0 {
        log.Printf("No placed bets found for event %s to settle.", eventID) 
        return &errors.ErrorNotFound{Entity:"Placed Bets for Event", ID: eventID} 
    }


	settleStatus := model.StatusLost
	if result == "win" {
		settleStatus = model.StatusWon
	}

	var firstError error 

	for _, bet := range betsToSettle {
		bet.Status = settleStatus
		err := s.repo.UpdateBet(bet) 
		if err != nil {
			log.Printf("Error settling bet ID %s for event %s: %v", bet.ID, eventID, err) 
			if firstError == nil {
				firstError = fmt.Errorf("failed to settle bet %s: %w", bet.ID, err)
			}
		} else {
             log.Printf("Bet ID %s settled successfully with status %s for event %s", bet.ID, settleStatus, eventID) 
        }
	}
    if firstError != nil {
        log.Printf("Finished settling event %s with errors.", eventID)
    } else {
        log.Printf("Successfully settled all placed bets for event %s.", eventID) 
    }

	return firstError 
}

// CreateUser handles the logic for creating a new user.
func (s *BetService) CreateUser(req *model.CreateUserRequest) (*model.User, error) {
	if err := req.Validate(); err != nil {
		log.Printf("Validation error creating user %s: %v", req.UserID, err)
		return nil, &errors.ErrorBadRequest{Message: fmt.Sprintf("validation failed: %s", err.Error())}
	}

	user := &model.User{
		ID: req.UserID,
	}

	createdUser, err := s.repo.CreateUser(user)
	if err != nil {
		log.Printf("Repository error creating user %s: %v", req.UserID, err)
        if _, ok := err.(*errors.ErrorConflict); ok {
            return nil, err
        }
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	log.Printf("User created successfully: ID=%s", createdUser.ID)
	return createdUser, nil
}

// GetUser retrieves a user by their ID.
func (s *BetService) GetUser(userID string) (*model.User, error) {
	if userID == "" {
		return nil, &errors.ErrorBadRequest{Message: "user ID cannot be empty"}
	}
	user, err := s.repo.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user %s: %v", userID, err)
		return nil, err
	}
	log.Printf("Retrieved user: ID=%s", user.ID)
	return user, nil
}

// ListUsers retrieves all users.
func (s *BetService) ListUsers() ([]*model.User, error) {
	users, err := s.repo.ListUsers()
	if err != nil {
		log.Printf("Error listing users: %v", err)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	log.Printf("Retrieved %d users", len(users))
	return users, nil
}

// UpdateUser handles updating user information.
func (s *BetService) UpdateUser(userID string, req *model.UpdateUserRequest) (*model.User, error) {
	if userID == "" {
		return nil, &errors.ErrorBadRequest{Message: "user ID cannot be empty"}
	}
	if err := req.Validate(); err != nil {
		log.Printf("Validation error updating user %s: %v", userID, err)
		return nil, &errors.ErrorBadRequest{Message: fmt.Sprintf("validation failed: %s", err.Error())}
	}

	// First, check if user exists
	userToUpdate, err := s.repo.GetUser(userID)
	if err != nil {
		log.Printf("Error finding user %s for update: %v", userID, err)
		return nil, err 
	}


	updatedUser, err := s.repo.UpdateUser(userToUpdate) 
	if err != nil {
		log.Printf("Repository error updating user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	log.Printf("User updated successfully: ID=%s", updatedUser.ID)
	return updatedUser, nil
}

// DeleteUser handles deleting a user.
func (s *BetService) DeleteUser(userID string) error {
	if userID == "" {
		return &errors.ErrorBadRequest{Message: "user ID cannot be empty"}
	}


	err := s.repo.DeleteUser(userID)
	if err != nil {
		log.Printf("Repository error deleting user %s: %v", userID, err)
		return err 
	}
	log.Printf("User deleted successfully: ID=%s", userID)
	return nil
}


// Modify GetUserBalance service method slightly
// It should rely on GetUser service method for consistency
func (s *BetService) GetUserBalance(userID string) (float64, error) {
	user, err := s.GetUser(userID) // Use the service GetUser method
	if err != nil {
        log.Printf("Error getting balance for user %s (via GetUser): %v", userID, err)
		return 0, err 
	}
	log.Printf("Retrieved balance for user %s: %.2f", userID, user.Balance)
	return user.Balance, nil
}
