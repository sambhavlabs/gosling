package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sourabhmandal/gosling/internal/domain"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
	"github.com/sourabhmandal/gosling/internal/domain/repository"
	"github.com/sourabhmandal/gosling/internal/domain/service"
)

// AuthResponseDTO represents the response returned upon successful authentication.
type AuthResponseDTO struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresAt   time.Time    `json:"expires_at"`
	IsNewUser   bool         `json:"is_new_user"`
	User        *entity.User `json:"user"`
}

// AuthUseCase defines application business logic for authentication and user management.
type AuthUseCase interface {
	LoginWithGoogleIDToken(ctx context.Context, idToken string) (*AuthResponseDTO, error)
	LoginWithGoogleCode(ctx context.Context, code string) (*AuthResponseDTO, error)
	GetGoogleAuthURL(state string) string
	GetUserProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error)
}

type authUseCase struct {
	userRepo       repository.UserRepository
	googleVerifier service.GoogleAuthVerifier
	tokenService   service.TokenService
}

// NewAuthUseCase creates an instance of AuthUseCase with injected dependencies.
func NewAuthUseCase(
	userRepo repository.UserRepository,
	googleVerifier service.GoogleAuthVerifier,
	tokenService service.TokenService,
) AuthUseCase {
	return &authUseCase{
		userRepo:       userRepo,
		googleVerifier: googleVerifier,
		tokenService:   tokenService,
	}
}

func (u *authUseCase) LoginWithGoogleIDToken(ctx context.Context, idToken string) (*AuthResponseDTO, error) {
	if idToken == "" {
		return nil, fmt.Errorf("%w: google id token is required", domain.ErrInvalidInput)
	}

	claims, err := u.googleVerifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrGoogleAuthFailed, err)
	}

	return u.processGoogleClaims(ctx, claims)
}

func (u *authUseCase) LoginWithGoogleCode(ctx context.Context, code string) (*AuthResponseDTO, error) {
	if code == "" {
		return nil, fmt.Errorf("%w: authorization code is required", domain.ErrInvalidInput)
	}

	claims, err := u.googleVerifier.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrGoogleAuthFailed, err)
	}

	return u.processGoogleClaims(ctx, claims)
}

func (u *authUseCase) processGoogleClaims(ctx context.Context, claims *entity.GoogleClaims) (*AuthResponseDTO, error) {
	var user *entity.User
	var isNewUser bool

	// 1. Try to find user by Google ID
	existingUser, err := u.userRepo.GetByGoogleID(ctx, claims.GoogleID)
	if err != nil && err != domain.ErrUserNotFound {
		return nil, fmt.Errorf("error querying user by google ID: %w", err)
	}

	if existingUser != nil {
		user = existingUser
		user.LastLoginAt = time.Now().UTC()
		if claims.Picture != "" && user.AvatarURL != claims.Picture {
			user.AvatarURL = claims.Picture
		}
		if claims.Name != "" && user.Name != claims.Name {
			user.Name = claims.Name
		}
		user.UpdatedAt = time.Now().UTC()
		if err := u.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update user login details: %w", err)
		}
	} else {
		// 2. If not found by Google ID, try to find user by Email
		existingEmailUser, err := u.userRepo.GetByEmail(ctx, claims.Email)
		if err != nil && err != domain.ErrUserNotFound {
			return nil, fmt.Errorf("error querying user by email: %w", err)
		}

		if existingEmailUser != nil {
			// Link Google ID to existing account
			user = existingEmailUser
			user.GoogleID = claims.GoogleID
			user.LastLoginAt = time.Now().UTC()
			if claims.Picture != "" && user.AvatarURL == "" {
				user.AvatarURL = claims.Picture
			}
			user.UpdatedAt = time.Now().UTC()
			if err := u.userRepo.Update(ctx, user); err != nil {
				return nil, fmt.Errorf("failed to link google account: %w", err)
			}
		} else {
			// 3. Implicit Registration: User does not exist, create new account
			newUser := entity.NewUser(claims.Email, claims.Name, claims.Picture, claims.GoogleID)
			if err := u.userRepo.Create(ctx, newUser); err != nil {
				return nil, fmt.Errorf("failed to implicitly register new user: %w", err)
			}
			user = newUser
			isNewUser = true
		}
	}

	// 4. Generate application JWT access token
	token, expiresAt, err := u.tokenService.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &AuthResponseDTO{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		IsNewUser:   isNewUser,
		User:        user,
	}, nil
}

func (u *authUseCase) GetGoogleAuthURL(state string) string {
	return u.googleVerifier.GetAuthURL(state)
}

func (u *authUseCase) GetUserProfile(ctx context.Context, userID uuid.UUID) (*entity.User, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}
