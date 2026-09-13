package security_test

import (
	"testing"
	"time"

	"github.com/sourabhmandal/gosling/internal/domain/entity"
	"github.com/sourabhmandal/gosling/internal/infrastructure/security"
)

func TestJWTService_GenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key-that-is-sufficiently-long-for-testing"
	duration := 2 * time.Hour
	jwtSvc := security.NewJWTService(secret, duration)

	user := entity.NewUser("test@example.com", "Test User", "https://avatar.com/pic.jpg", "google-12345")

	tokenString, expiresAt, err := jwtSvc.GenerateToken(user)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	if tokenString == "" {
		t.Fatal("expected non-empty token string")
	}

	if expiresAt.Before(time.Now()) {
		t.Fatal("expected expiresAt to be in the future")
	}

	// Validate valid token
	claims, err := jwtSvc.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("unexpected error validating valid token: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("expected user ID %v, got %v", user.ID, claims.UserID)
	}

	if claims.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, claims.Email)
	}

	if claims.Name != user.Name {
		t.Errorf("expected name %s, got %s", user.Name, claims.Name)
	}
}

func TestJWTService_InvalidToken(t *testing.T) {
	jwtSvc := security.NewJWTService("secret-1", time.Hour)
	otherSvc := security.NewJWTService("secret-2", time.Hour)

	user := entity.NewUser("hacker@example.com", "Hacker", "", "google-999")
	token, _, _ := otherSvc.GenerateToken(user)

	_, err := jwtSvc.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error validating token signed with different secret")
	}
}
