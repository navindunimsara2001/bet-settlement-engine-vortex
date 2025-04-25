package model

import (
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Balance   float64   `json:"balance"` 
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUserRequest defines the payload for creating a new user.
type CreateUserRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Name string `json:"name" validate:"required"`
}

func (req *CreateUserRequest) Validate() error {
	return validate.Struct(req)
}

// UpdateUserRequest defines the payload for updating user details.
type UpdateUserRequest struct {
}

func (req *UpdateUserRequest) Validate() error {
	// Add validation rules if needed for updatable fields
	return validate.Struct(req)
}