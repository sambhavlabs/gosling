package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sourabhmandal/gosling/internal/domain"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
	"github.com/sourabhmandal/gosling/internal/domain/service"
)

type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secretKey     []byte
	tokenDuration time.Duration
	issuer        string
}

// NewJWTService creates a new TokenService backed by HMAC SHA-256 JWTs.
func NewJWTService(secretKey string, tokenDuration time.Duration) service.TokenService {
	return &jwtService{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
		issuer:        "gosling-api",
	}
}

func (s *jwtService) GenerateToken(user *entity.User) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.tokenDuration)

	claims := &jwtClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

func (s *jwtService) ValidateToken(tokenString string) (*entity.UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	parsedUUID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, errors.New("invalid user ID in token claims")
	}

	return &entity.UserClaims{
		UserID: parsedUUID,
		Email:  claims.Email,
		Name:   claims.Name,
	}, nil
}
