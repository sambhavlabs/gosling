package service

import (
	"context"
	"time"

	"github.com/sourabhmandal/gosling/internal/domain/entity"
)

// GoogleAuthVerifier defines the external port for Google OAuth / ID token verification.
type GoogleAuthVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*entity.GoogleClaims, error)
	ExchangeCode(ctx context.Context, authCode string) (*entity.GoogleClaims, error)
	GetAuthURL(state string) string
}

// TokenService defines the port for application JWT generation and validation.
type TokenService interface {
	GenerateToken(user *entity.User) (token string, expiresAt time.Time, err error)
	ValidateToken(tokenString string) (*entity.UserClaims, error)
}
