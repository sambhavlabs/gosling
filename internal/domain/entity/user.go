package entity

import (
	"time"

	"github.com/google/uuid"
)

// User represents the core domain user entity.
type User struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	AvatarURL   string    `json:"avatar_url"`
	GoogleID    string    `json:"google_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastLoginAt time.Time `json:"last_login_at"`
}

// NewUser creates a new domain User instance with generated UUID and timestamps.
func NewUser(email, name, avatarURL, googleID string) *User {
	now := time.Now().UTC()
	return &User{
		ID:          uuid.New(),
		Email:       email,
		Name:        name,
		AvatarURL:   avatarURL,
		GoogleID:    googleID,
		CreatedAt:   now,
		UpdatedAt:   now,
		LastLoginAt: now,
	}
}

// GoogleClaims holds the verified claims returned from Google OAuth/ID token.
type GoogleClaims struct {
	GoogleID      string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

// UserClaims represents the JWT claims payload for authenticated sessions.
type UserClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Name   string    `json:"name"`
}
