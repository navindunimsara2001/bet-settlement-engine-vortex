package model

import (
	"github.com/go-playground/validator/v10"
	"time"
)

var validate = validator.New()

type BetStatus string

const (
	StatusPlaced BetStatus = "PLACED"
	StatusWon    BetStatus = "WON"
	StatusLost   BetStatus = "LOST"
)

type Bet struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id" validate:"required"`
	EventID   string    `json:"event_id" validate:"required"`
	Odds      float64   `json:"odds" validate:"required,gt=1"`
	Amount    float64   `json:"amount" validate:"required,gt=0"`
	Status    BetStatus `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	SettledAt time.Time `json:"settled_at,omitempty"`
}

type PlaceBetRequest struct {
	UserID  string  `json:"user_id" validate:"required"`
	EventID string  `json:"event_id" validate:"required"`
	Odds    float64 `json:"odds" validate:"required,gt=1"`
	Amount  float64 `json:"amount" validate:"required,gt=0"`
}

func (p *PlaceBetRequest) Validate() error {
	return validate.Struct(p)
}

type SettleBetRequest struct {
	Result string `json:"result" validate:"required,oneof=win lose"`
}

func (s *SettleBetRequest) Validate() error {
	return validate.Struct(s)
}