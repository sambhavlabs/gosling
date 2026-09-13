package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sourabhmandal/gosling/internal/domain/entity"
	"github.com/sourabhmandal/gosling/internal/domain/service"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

type googleAuthService struct {
	clientID     string
	clientSecret string
	redirectURL  string
	oauthConfig  *oauth2.Config
	httpClient   *http.Client
}

// NewGoogleAuthService creates a new GoogleAuthVerifier.
func NewGoogleAuthService(clientID, clientSecret, redirectURL string) service.GoogleAuthVerifier {
	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"openid",
		},
		Endpoint: google.Endpoint,
	}

	return &googleAuthService{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		oauthConfig:  oauthConfig,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// VerifyIDToken validates a Google ID Token (JWT) sent directly by frontends.
func (s *googleAuthService) VerifyIDToken(ctx context.Context, idTokenString string) (*entity.GoogleClaims, error) {
	if idTokenString == "" {
		return nil, errors.New("empty id token")
	}

	// 1. First attempt: Official Google ID Token validation library
	payload, err := idtoken.Validate(ctx, idTokenString, s.clientID)
	if err == nil && payload != nil {
		claims := &entity.GoogleClaims{
			GoogleID: payload.Subject,
		}
		if email, ok := payload.Claims["email"].(string); ok {
			claims.Email = email
		}
		if emailVerified, ok := payload.Claims["email_verified"].(bool); ok {
			claims.EmailVerified = emailVerified
		}
		if name, ok := payload.Claims["name"].(string); ok {
			claims.Name = name
		}
		if picture, ok := payload.Claims["picture"].(string); ok {
			claims.Picture = picture
		}
		if givenName, ok := payload.Claims["given_name"].(string); ok {
			claims.GivenName = givenName
		}
		if familyName, ok := payload.Claims["family_name"].(string); ok {
			claims.FamilyName = familyName
		}
		return claims, nil
	}

	// 2. Secondary fallback: Google TokenInfo REST API endpoint (for development or audience-agnostic verification)
	reqURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idTokenString)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokeninfo request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call google tokeninfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid google token: status %d (%s)", resp.StatusCode, string(body))
	}

	var rawClaims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Aud           string `json:"aud"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawClaims); err != nil {
		return nil, fmt.Errorf("failed to decode google tokeninfo response: %w", err)
	}

	if rawClaims.Sub == "" || rawClaims.Email == "" {
		return nil, errors.New("incomplete claims returned from google tokeninfo")
	}

	return &entity.GoogleClaims{
		GoogleID:      rawClaims.Sub,
		Email:         rawClaims.Email,
		EmailVerified: rawClaims.EmailVerified == "true",
		Name:          rawClaims.Name,
		Picture:       rawClaims.Picture,
		GivenName:     rawClaims.GivenName,
		FamilyName:    rawClaims.FamilyName,
	}, nil
}

// ExchangeCode exchanges an authorization code from OAuth2 callback for user details.
func (s *googleAuthService) ExchangeCode(ctx context.Context, authCode string) (*entity.GoogleClaims, error) {
	token, err := s.oauthConfig.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange oauth code: %w", err)
	}

	client := s.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch google user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve user info: status code %d", resp.StatusCode)
	}

	var userInfo struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode google user info: %w", err)
	}

	return &entity.GoogleClaims{
		GoogleID:      userInfo.ID,
		Email:         userInfo.Email,
		EmailVerified: userInfo.VerifiedEmail,
		Name:          userInfo.Name,
		Picture:       userInfo.Picture,
		GivenName:     userInfo.GivenName,
		FamilyName:    userInfo.FamilyName,
	}, nil
}

// GetAuthURL builds the Google OAuth2 consent URL.
func (s *googleAuthService) GetAuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}
