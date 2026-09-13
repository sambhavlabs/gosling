package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sourabhmandal/gosling/internal/domain"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
	"github.com/sourabhmandal/gosling/internal/usecase"
)

// MockUserRepository implements repository.UserRepository in memory
type MockUserRepository struct {
	users map[uuid.UUID]*entity.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{users: make(map[uuid.UUID]*entity.User)}
}

func (m *MockUserRepository) Create(ctx context.Context, user *entity.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *MockUserRepository) GetByGoogleID(ctx context.Context, googleID string) (*entity.User, error) {
	for _, u := range m.users {
		if u.GoogleID == googleID {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *MockUserRepository) Update(ctx context.Context, user *entity.User) error {
	if _, ok := m.users[user.ID]; !ok {
		return domain.ErrUserNotFound
	}
	m.users[user.ID] = user
	return nil
}

// MockGoogleAuthVerifier implements service.GoogleAuthVerifier
type MockGoogleAuthVerifier struct {
	claimsToReturn *entity.GoogleClaims
	errToReturn    error
}

func (m *MockGoogleAuthVerifier) VerifyIDToken(ctx context.Context, idToken string) (*entity.GoogleClaims, error) {
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	return m.claimsToReturn, nil
}

func (m *MockGoogleAuthVerifier) ExchangeCode(ctx context.Context, authCode string) (*entity.GoogleClaims, error) {
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	return m.claimsToReturn, nil
}

func (m *MockGoogleAuthVerifier) GetAuthURL(state string) string {
	return "https://accounts.google.com/o/oauth2/v2/auth?state=" + state
}

// MockTokenService implements service.TokenService
type MockTokenService struct{}

func (m *MockTokenService) GenerateToken(user *entity.User) (string, time.Time, error) {
	return "mock-jwt-token", time.Now().Add(24 * time.Hour), nil
}

func (m *MockTokenService) ValidateToken(tokenString string) (*entity.UserClaims, error) {
	return &entity.UserClaims{
		UserID: uuid.New(),
		Email:  "test@example.com",
		Name:   "Test User",
	}, nil
}

func TestAuthUseCase_ImplicitRegistration(t *testing.T) {
	repo := NewMockUserRepository()
	verifier := &MockGoogleAuthVerifier{
		claimsToReturn: &entity.GoogleClaims{
			GoogleID:      "google-unique-987",
			Email:         "newuser@example.com",
			EmailVerified: true,
			Name:          "New User",
			Picture:       "https://avatar.com/pic.jpg",
		},
	}
	tokenSvc := &MockTokenService{}

	authUC := usecase.NewAuthUseCase(repo, verifier, tokenSvc)

	result, err := authUC.LoginWithGoogleIDToken(context.Background(), "valid-google-id-token")
	if err != nil {
		t.Fatalf("unexpected error during implicit registration: %v", err)
	}

	if !result.IsNewUser {
		t.Errorf("expected is_new_user to be true for new user")
	}

	if result.User.Email != "newuser@example.com" {
		t.Errorf("expected email to be newuser@example.com, got %s", result.User.Email)
	}

	if result.AccessToken != "mock-jwt-token" {
		t.Errorf("expected mock token, got %s", result.AccessToken)
	}

	// Verify subsequent login does not register a new user
	secondResult, err := authUC.LoginWithGoogleIDToken(context.Background(), "valid-google-id-token")
	if err != nil {
		t.Fatalf("unexpected error during second login: %v", err)
	}

	if secondResult.IsNewUser {
		t.Errorf("expected is_new_user to be false for existing user")
	}

	if secondResult.User.ID != result.User.ID {
		t.Errorf("expected existing user ID to match")
	}
}
