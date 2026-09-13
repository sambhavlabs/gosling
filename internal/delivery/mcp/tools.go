package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sourabhmandal/gosling/internal/domain"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
)

// --- Tool Inputs & Outputs (DTOs) ---

// UserDTO represents a JSON-Schema compliant user entity representation.
type UserDTO struct {
	ID          string `json:"id" jsonschema:"User UUID string"`
	Email       string `json:"email" jsonschema:"User email address"`
	Name        string `json:"name" jsonschema:"User display name"`
	AvatarURL   string `json:"avatar_url" jsonschema:"Avatar image URL"`
	GoogleID    string `json:"google_id,omitempty" jsonschema:"Linked Google account ID"`
	CreatedAt   string `json:"created_at" jsonschema:"Account creation timestamp in RFC3339 format"`
	UpdatedAt   string `json:"updated_at" jsonschema:"Account last update timestamp in RFC3339 format"`
	LastLoginAt string `json:"last_login_at" jsonschema:"Last login timestamp in RFC3339 format"`
}

func toUserDTO(u *entity.User) *UserDTO {
	if u == nil {
		return nil
	}
	return &UserDTO{
		ID:          u.ID.String(),
		Email:       u.Email,
		Name:        u.Name,
		AvatarURL:   u.AvatarURL,
		GoogleID:    u.GoogleID,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339),
		LastLoginAt: u.LastLoginAt.Format(time.RFC3339),
	}
}

// ChatHistoryDTO represents a JSON-Schema compliant chat history record.
type ChatHistoryDTO struct {
	ID           string `json:"id" jsonschema:"History record UUID string"`
	UserID       string `json:"user_id,omitempty" jsonschema:"User UUID string"`
	Model        string `json:"model" jsonschema:"LLM model used"`
	Prompt       string `json:"prompt" jsonschema:"User prompt text"`
	Response     string `json:"response" jsonschema:"LLM generated response text"`
	PromptTokens int32  `json:"prompt_tokens" jsonschema:"Prompt token count"`
	OutputTokens int32  `json:"output_tokens" jsonschema:"Output token count"`
	CreatedAt    string `json:"created_at" jsonschema:"Timestamp of the chat in RFC3339 format"`
}

func toChatHistoryDTO(h *entity.ChatHistory) *ChatHistoryDTO {
	if h == nil {
		return nil
	}
	var uid string
	if h.UserID != nil {
		uid = h.UserID.String()
	}
	return &ChatHistoryDTO{
		ID:           h.ID.String(),
		UserID:       uid,
		Model:        h.Model,
		Prompt:       h.Prompt,
		Response:     h.Response,
		PromptTokens: h.PromptTokens,
		OutputTokens: h.OutputTokens,
		CreatedAt:    h.CreatedAt.Format(time.RFC3339),
	}
}

// ChatToolInput defines the parameters for the chat LLM tool.
type ChatToolInput struct {
	Prompt            string               `json:"prompt,omitempty" jsonschema:"The user prompt or instruction to send to the LLM"`
	Messages          []entity.ChatMessage `json:"messages,omitempty" jsonschema:"Multi-turn conversation messages array"`
	Model             string               `json:"model,omitempty" jsonschema:"Gemini model name to use (e.g. gemini-2.5-flash, gemini-2.5-pro, gemini-2.0-flash)"`
	SystemInstruction string               `json:"system_instruction,omitempty" jsonschema:"Optional system prompt instructions for behavior conditioning"`
	Temperature       *float32             `json:"temperature,omitempty" jsonschema:"Randomness parameter from 0.0 to 2.0"`
	MaxOutputTokens   *int32               `json:"max_output_tokens,omitempty" jsonschema:"Maximum tokens to generate"`
	AuthToken         string               `json:"auth_token,omitempty" jsonschema:"Optional JWT authentication Bearer token to associate chat history with user account"`
	UserID            string               `json:"user_id,omitempty" jsonschema:"Optional user UUID string to associate chat history"`
}

// ChatToolOutput defines the response format for the chat LLM tool.
type ChatToolOutput struct {
	ResponseText    string `json:"response_text" jsonschema:"LLM generated text output"`
	ModelUsed       string `json:"model_used" jsonschema:"Gemini model name used"`
	PromptTokens    int32  `json:"prompt_tokens" jsonschema:"Number of prompt tokens"`
	CandidateTokens int32  `json:"candidate_tokens" jsonschema:"Number of candidate tokens generated"`
	TotalTokens     int32  `json:"total_tokens" jsonschema:"Total tokens utilized"`
	CreatedAt       string `json:"created_at" jsonschema:"ISO timestamp when response was generated"`
}

// GetAvailableModelsInput defines the parameters for listing models.
type GetAvailableModelsInput struct{}

// GetAvailableModelsOutput defines the output for listing models.
type GetAvailableModelsOutput struct {
	DefaultModel string   `json:"default_model" jsonschema:"Default recommended Gemini model"`
	Models       []string `json:"models" jsonschema:"List of available Gemini model identifiers"`
}

// GetChatHistoryInput defines the parameters for querying user chat history.
type GetChatHistoryInput struct {
	UserID    string `json:"user_id,omitempty" jsonschema:"User UUID whose history to fetch"`
	AuthToken string `json:"auth_token,omitempty" jsonschema:"JWT authentication token to identify user"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum number of history records to return (default 20)"`
}

// GetChatHistoryOutput defines the output for querying user chat history.
type GetChatHistoryOutput struct {
	UserID  string            `json:"user_id" jsonschema:"Target user UUID"`
	Count   int               `json:"count" jsonschema:"Number of history records returned"`
	History []*ChatHistoryDTO `json:"history" jsonschema:"Array of chat history items"`
}

// VerifyGoogleIDTokenInput defines the parameters for verifying a Google ID Token.
type VerifyGoogleIDTokenInput struct {
	IDToken string `json:"id_token" jsonschema:"Google OAuth ID Token to verify and login/register user"`
}

// LoginWithGoogleCodeInput defines the parameters for exchanging a Google OAuth code.
type LoginWithGoogleCodeInput struct {
	Code string `json:"code" jsonschema:"Google OAuth authorization code to exchange for user session"`
}

// AuthToolOutput defines the response format for authentication tools.
type AuthToolOutput struct {
	AccessToken string   `json:"access_token" jsonschema:"JWT Bearer access token for authenticated session"`
	TokenType   string   `json:"token_type" jsonschema:"Token type (Bearer)"`
	ExpiresAt   string   `json:"expires_at" jsonschema:"Token expiration ISO timestamp"`
	IsNewUser   bool     `json:"is_new_user" jsonschema:"Whether user was registered for the first time"`
	User        *UserDTO `json:"user" jsonschema:"Authenticated user profile details"`
}

// GetGoogleAuthURLInput defines the parameters for generating Google OAuth URL.
type GetGoogleAuthURLInput struct {
	State string `json:"state,omitempty" jsonschema:"Optional OAuth state parameter"`
}

// GetGoogleAuthURLOutput defines the output for generating Google OAuth URL.
type GetGoogleAuthURLOutput struct {
	AuthURL string `json:"auth_url" jsonschema:"Google OAuth consent screen URL"`
}

// GetUserProfileInput defines the parameters for looking up a user profile.
type GetUserProfileInput struct {
	UserID    string `json:"user_id,omitempty" jsonschema:"User UUID to look up"`
	AuthToken string `json:"auth_token,omitempty" jsonschema:"JWT authentication token to look up profile"`
}

// GetUserProfileOutput defines the output for looking up a user profile.
type GetUserProfileOutput struct {
	User *UserDTO `json:"user" jsonschema:"User profile details"`
}

// HealthCheckInput defines the input for health checking.
type HealthCheckInput struct{}

// HealthCheckOutput defines the output for health checking.
type HealthCheckOutput struct {
	Status      string   `json:"status" jsonschema:"Server health status"`
	Service     string   `json:"service" jsonschema:"Service identifier"`
	Environment string   `json:"environment" jsonschema:"Operational environment mode"`
	Version     string   `json:"version" jsonschema:"Application version"`
	Transports  []string `json:"transports" jsonschema:"Supported MCP transports"`
	Timestamp   string   `json:"timestamp" jsonschema:"Current server timestamp"`
}

// --- Tool Implementations ---

func (s *MCPServer) registerTools() {
	// 1. Tool: chat
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "chat",
		Description: "Generate LLM responses using Gemini models via Google GenAI SDK. Supports single prompt or multi-turn messages, system instructions, temperature, max tokens, and optional user attribution.",
	}, s.handleChatTool)

	// 2. Tool: get_available_models
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_available_models",
		Description: "List all supported and recommended Gemini LLM models available on the Gosling MCP server.",
	}, s.handleGetAvailableModelsTool)

	// 3. Tool: get_chat_history
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_chat_history",
		Description: "Retrieve past chat history records for an authenticated user by User ID or JWT Auth Token.",
	}, s.handleGetChatHistoryTool)

	// 4. Tool: verify_google_id_token
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "verify_google_id_token",
		Description: "Authenticate or implicitly register a user via a Google ID token. Returns JWT access token and user profile.",
	}, s.handleVerifyGoogleIDTokenTool)

	// 5. Tool: login_with_google_code
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "login_with_google_code",
		Description: "Authenticate or implicitly register a user via Google OAuth authorization code exchange.",
	}, s.handleLoginWithGoogleCodeTool)

	// 6. Tool: get_google_auth_url
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_google_auth_url",
		Description: "Generate the Google OAuth2 consent screen URL for browser authentication flow.",
	}, s.handleGetGoogleAuthURLTool)

	// 7. Tool: get_user_profile
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_user_profile",
		Description: "Fetch user profile details by User UUID or JWT Auth Token.",
	}, s.handleGetUserProfileTool)

	// 8. Tool: health_check
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "health_check",
		Description: "Check server status, environment, version, and supported MCP transports (JSON-RPC 2.0 Stdio, SSE, Streamable HTTP).",
	}, s.handleHealthCheckTool)
}

func (s *MCPServer) handleChatTool(ctx context.Context, req *mcp.CallToolRequest, input ChatToolInput) (*mcp.CallToolResult, *ChatToolOutput, error) {
	var targetUserID *uuid.UUID

	// Resolve user ID if AuthToken or UserID is supplied
	if input.AuthToken != "" && s.cfg.TokenService != nil {
		tokenStr := strings.TrimPrefix(input.AuthToken, "Bearer ")
		tokenStr = strings.TrimSpace(tokenStr)
		if claims, err := s.cfg.TokenService.ValidateToken(tokenStr); err == nil && claims != nil {
			targetUserID = &claims.UserID
		}
	} else if input.UserID != "" {
		if parsedID, err := uuid.Parse(input.UserID); err == nil {
			targetUserID = &parsedID
		}
	}

	chatReq := &entity.ChatRequest{
		Model:             input.Model,
		Prompt:            input.Prompt,
		Messages:          input.Messages,
		SystemInstruction: input.SystemInstruction,
		Temperature:       input.Temperature,
		MaxOutputTokens:   input.MaxOutputTokens,
	}

	resp, err := s.cfg.ChatUseCase.SendMessage(ctx, targetUserID, chatReq)
	if err != nil {
		return nil, nil, fmt.Errorf("chat generation failed: %w", err)
	}

	out := &ChatToolOutput{
		ResponseText:    resp.ResponseText,
		ModelUsed:       resp.ModelUsed,
		PromptTokens:    resp.PromptTokens,
		CandidateTokens: resp.CandidateTokens,
		TotalTokens:     resp.TotalTokens,
		CreatedAt:       resp.CreatedAt.Format(time.RFC3339),
	}
	return nil, out, nil
}

func (s *MCPServer) handleGetAvailableModelsTool(ctx context.Context, req *mcp.CallToolRequest, input GetAvailableModelsInput) (*mcp.CallToolResult, *GetAvailableModelsOutput, error) {
	models, err := s.cfg.ChatUseCase.GetAvailableModels(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get available models: %w", err)
	}

	out := &GetAvailableModelsOutput{
		DefaultModel: "gemini-2.5-flash",
		Models:       models,
	}
	return nil, out, nil
}

func (s *MCPServer) handleGetChatHistoryTool(ctx context.Context, req *mcp.CallToolRequest, input GetChatHistoryInput) (*mcp.CallToolResult, *GetChatHistoryOutput, error) {
	var targetUserID uuid.UUID
	var found bool

	if input.AuthToken != "" && s.cfg.TokenService != nil {
		tokenStr := strings.TrimPrefix(input.AuthToken, "Bearer ")
		tokenStr = strings.TrimSpace(tokenStr)
		if claims, err := s.cfg.TokenService.ValidateToken(tokenStr); err == nil && claims != nil {
			targetUserID = claims.UserID
			found = true
		}
	}

	if !found && input.UserID != "" {
		if parsed, err := uuid.Parse(input.UserID); err == nil {
			targetUserID = parsed
			found = true
		}
	}

	if !found {
		return nil, nil, fmt.Errorf("%w: a valid user_id or auth_token is required to retrieve chat history", domain.ErrInvalidInput)
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	history, err := s.cfg.ChatUseCase.GetUserChatHistory(ctx, targetUserID, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chat history: %w", err)
	}

	dtos := make([]*ChatHistoryDTO, 0, len(history))
	for _, h := range history {
		dtos = append(dtos, toChatHistoryDTO(h))
	}

	out := &GetChatHistoryOutput{
		UserID:  targetUserID.String(),
		Count:   len(dtos),
		History: dtos,
	}
	return nil, out, nil
}

func (s *MCPServer) handleVerifyGoogleIDTokenTool(ctx context.Context, req *mcp.CallToolRequest, input VerifyGoogleIDTokenInput) (*mcp.CallToolResult, *AuthToolOutput, error) {
	if s.cfg.AuthUseCase == nil {
		return nil, nil, fmt.Errorf("auth usecase is not configured")
	}

	res, err := s.cfg.AuthUseCase.LoginWithGoogleIDToken(ctx, input.IDToken)
	if err != nil {
		return nil, nil, fmt.Errorf("google id token authentication failed: %w", err)
	}

	out := &AuthToolOutput{
		AccessToken: res.AccessToken,
		TokenType:   res.TokenType,
		ExpiresAt:   res.ExpiresAt.Format(time.RFC3339),
		IsNewUser:   res.IsNewUser,
		User:        toUserDTO(res.User),
	}
	return nil, out, nil
}

func (s *MCPServer) handleLoginWithGoogleCodeTool(ctx context.Context, req *mcp.CallToolRequest, input LoginWithGoogleCodeInput) (*mcp.CallToolResult, *AuthToolOutput, error) {
	if s.cfg.AuthUseCase == nil {
		return nil, nil, fmt.Errorf("auth usecase is not configured")
	}

	res, err := s.cfg.AuthUseCase.LoginWithGoogleCode(ctx, input.Code)
	if err != nil {
		return nil, nil, fmt.Errorf("google authorization code login failed: %w", err)
	}

	out := &AuthToolOutput{
		AccessToken: res.AccessToken,
		TokenType:   res.TokenType,
		ExpiresAt:   res.ExpiresAt.Format(time.RFC3339),
		IsNewUser:   res.IsNewUser,
		User:        toUserDTO(res.User),
	}
	return nil, out, nil
}

func (s *MCPServer) handleGetGoogleAuthURLTool(ctx context.Context, req *mcp.CallToolRequest, input GetGoogleAuthURLInput) (*mcp.CallToolResult, *GetGoogleAuthURLOutput, error) {
	if s.cfg.AuthUseCase == nil {
		return nil, nil, fmt.Errorf("auth usecase is not configured")
	}

	state := input.State
	if state == "" {
		state = "gosling-mcp-state"
	}

	url := s.cfg.AuthUseCase.GetGoogleAuthURL(state)
	return nil, &GetGoogleAuthURLOutput{AuthURL: url}, nil
}

func (s *MCPServer) handleGetUserProfileTool(ctx context.Context, req *mcp.CallToolRequest, input GetUserProfileInput) (*mcp.CallToolResult, *GetUserProfileOutput, error) {
	if s.cfg.AuthUseCase == nil {
		return nil, nil, fmt.Errorf("auth usecase is not configured")
	}

	var targetUserID uuid.UUID
	var found bool

	if input.AuthToken != "" && s.cfg.TokenService != nil {
		tokenStr := strings.TrimPrefix(input.AuthToken, "Bearer ")
		tokenStr = strings.TrimSpace(tokenStr)
		if claims, err := s.cfg.TokenService.ValidateToken(tokenStr); err == nil && claims != nil {
			targetUserID = claims.UserID
			found = true
		}
	}

	if !found && input.UserID != "" {
		if parsed, err := uuid.Parse(input.UserID); err == nil {
			targetUserID = parsed
			found = true
		}
	}

	if !found {
		return nil, nil, fmt.Errorf("%w: a valid user_id or auth_token is required to view profile", domain.ErrInvalidInput)
	}

	user, err := s.cfg.AuthUseCase.GetUserProfile(ctx, targetUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	return nil, &GetUserProfileOutput{User: toUserDTO(user)}, nil
}

func (s *MCPServer) handleHealthCheckTool(ctx context.Context, req *mcp.CallToolRequest, input HealthCheckInput) (*mcp.CallToolResult, *HealthCheckOutput, error) {
	out := &HealthCheckOutput{
		Status:      "healthy",
		Service:     s.cfg.AppName,
		Environment: s.cfg.AppEnv,
		Version:     s.cfg.AppVersion,
		Transports:  []string{"stdio", "streamable-http", "sse"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	return nil, out, nil
}
