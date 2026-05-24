package auth

import "time"

// User represents the internal system user database model.
type User struct {
	ID           int       `json:"id" example:"1"`
	Username     string    `json:"username" example:"john"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at" example:"2026-05-22T12:00:00Z"`
}

// APIKey represents the structure of an API key used for machine-to-machine authentication.
type APIKey struct {
	ID        int       `json:"id" example:"1"`
	UserID    int       `json:"user_id" example:"1"`
	KeyHash   string    `json:"-"`
	Name      string    `json:"name" example:"cli-token"`
	CreatedAt time.Time `json:"created_at" example:"2026-05-22T12:00:00Z"`
	ExpiresAt time.Time `json:"expires_at" example:"2027-05-22T12:00:00Z"`
}

// LoginRequest represents the user credentials payload.
// @Name LoginRequest
type LoginRequest struct {
	// @Param username example true "john"
	Username string `json:"username" binding:"required" example:"john"`
	// @Param password example true "supersecure123"
	Password string `json:"password" binding:"required" example:"supersecure123"`
}

// LoginResponse represents the successful authentication response.
// @Name LoginResponse
type LoginResponse struct {
	Token    string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Username string `json:"username" example:"john"`
}
