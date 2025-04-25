package memory

import (
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/internal/model"
	"github.com/navindunimsara2001/bet-settlement-engine-vortex/pkg/errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryBetRepository stores bets and user balances in memory.
// It uses mutexes for concurrency safety[cite: 4].
type InMemoryBetRepository struct {
	mu      sync.RWMutex
	bets    map[string]*model.Bet   
	betsByEvent map[string][]*model.Bet 
	users   map[string]*model.User  
}

// NewInMemoryBetRepository creates a new in-memory repository.
func NewInMemoryBetRepository() *InMemoryBetRepository {
	return &InMemoryBetRepository{
		bets:    make(map[string]*model.Bet),
		betsByEvent: make(map[string][]*model.Bet),
		users:   make(map[string]*model.User),
	}
}

// PlaceBet stores a new bet and updates the user's balance.
func (r *InMemoryBetRepository) PlaceBet(bet *model.Bet) (*model.Bet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[bet.UserID]
	if !exists {
		// Or create user on the fly
		return nil, &errors.ErrorNotFound{Entity: "User", ID: bet.UserID}
	}

	if user.Balance < bet.Amount {
		return nil, &errors.ErrorBadRequest{Message: fmt.Sprintf("insufficient balance: current %.2f, required %.2f", user.Balance, bet.Amount)}
	}

	bet.ID = uuid.New().String() 
	bet.Status = model.StatusPlaced
	bet.CreatedAt = time.Now()

	r.bets[bet.ID] = bet
	r.betsByEvent[bet.EventID] = append(r.betsByEvent[bet.EventID], bet)


	// Deduct amount from user balance
	user.Balance -= bet.Amount
	user.UpdatedAt = time.Now()
	r.users[user.ID] = user 


	return bet, nil
}

// FindBetsByEvent retrieves all bets for a specific event that are not yet settled.
func (r *InMemoryBetRepository) FindBetsByEvent(eventID string) ([]*model.Bet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bets, exists := r.betsByEvent[eventID]
	if !exists {
		return []*model.Bet{}, nil 
	}

	// Filter for placed bets only
	placedBets := []*model.Bet{}
	for _, b := range bets {
		if b.Status == model.StatusPlaced {
			placedBets = append(placedBets, b)
		}
	}

	if len(placedBets) == 0 {
        return []*model.Bet{}, nil
	}

	return placedBets, nil
}

// UpdateBet updates the status and settlement time of a bet.
func (r *InMemoryBetRepository) UpdateBet(bet *model.Bet) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existingBet, exists := r.bets[bet.ID]
	if !exists {
		return &errors.ErrorNotFound{Entity: "Bet", ID: bet.ID}
	}

	// Only allow updating status if it's currently PLACED
	if existingBet.Status != model.StatusPlaced {
		return &errors.ErrorConflict{Message: fmt.Sprintf("bet %s already settled with status %s", bet.ID, existingBet.Status)}
	}


	existingBet.Status = bet.Status
	existingBet.SettledAt = time.Now()
	r.bets[bet.ID] = existingBet 


	// Update user balance if the bet won
	if bet.Status == model.StatusWon {
		user, userExists := r.users[existingBet.UserID]
		if !userExists {
			return fmt.Errorf("internal error: user %s not found for winning bet %s", existingBet.UserID, bet.ID)
		}
		payout := existingBet.Amount * existingBet.Odds
		user.Balance += payout
		user.UpdatedAt = time.Now()
		r.users[user.ID] = user 
	}


	return nil
}

// CreateUser adds a new user to the repository.
func (r *InMemoryBetRepository) CreateUser(user *model.User) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; exists {
		return nil, &errors.ErrorConflict{Message: fmt.Sprintf("user with ID '%s' already exists", user.ID)}
	}

	// Set defaults
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.Balance == 0 {
		user.Balance = 1000.0
	}


	r.users[user.ID] = user
	return user, nil
}

// GetUser retrieves a specific user by ID.
func (r *InMemoryBetRepository) GetUser(userID string) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[userID]
	if !exists {
		return nil, &errors.ErrorNotFound{Entity: "User", ID: userID}
	}
	return user, nil
}

// ListUsers retrieves all users.
func (r *InMemoryBetRepository) ListUsers() ([]*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userList := make([]*model.User, 0, len(r.users))
	for _, user := range r.users {
		userList = append(userList, user)
	}
	return userList, nil
}

// UpdateUser updates details of an existing user.
func (r *InMemoryBetRepository) UpdateUser(user *model.User) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existingUser, exists := r.users[user.ID]
	if !exists {
		return nil, &errors.ErrorNotFound{Entity: "User", ID: user.ID}
	}

	existingUser.UpdatedAt = time.Now()
	r.users[user.ID] = existingUser

	return existingUser, nil
}

// DeleteUser removes a user from the repository.
func (r *InMemoryBetRepository) DeleteUser(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[userID]; !exists {
		return &errors.ErrorNotFound{Entity: "User", ID: userID}
	}
	delete(r.users, userID)
	return nil
}


// Modify FindOrCreateUser to use the new CreateUser logic
func (r *InMemoryBetRepository) FindOrCreateUser(userID string) (*model.User, error) {
    r.mu.RLock()
    user, exists := r.users[userID]
    r.mu.RUnlock() 

    if exists {
        return user, nil
    }

    // If not found, attempt to create
    newUser := &model.User{ID: userID}
    createdUser, err := r.CreateUser(newUser)
    if err != nil {
        if _, ok := err.(*errors.ErrorConflict); ok {
            // User was created by another request, try getting it again
             r.mu.RLock()
             user, exists = r.users[userID]
             r.mu.RUnlock()
             if exists {
                 return user, nil
             }
             return nil, fmt.Errorf("failed to find or create user '%s' after conflict", userID)
        }
        // Other creation error
        return nil, fmt.Errorf("failed to create user '%s': %w", userID, err)
    }
    return createdUser, nil
}


// Modify GetUserBalance to rely on GetUser
func (r *InMemoryBetRepository) GetUserBalance(userID string) (float64, error) {
	user, err := r.GetUser(userID) // Use the GetUser method
	if err != nil {
		return 0, err
	}
	return user.Balance, nil
}