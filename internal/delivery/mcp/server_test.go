package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	deliveryMCP "github.com/sourabhmandal/gosling/internal/delivery/mcp"
	"github.com/sourabhmandal/gosling/internal/domain"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
	"github.com/sourabhmandal/gosling/internal/usecase"
)

// --- Test Mocks ---

type MockLLMService struct {
	responseToReturn *entity.ChatResponse
}

func (m *MockLLMService) GenerateContent(ctx context.Context, req *entity.ChatRequest) (*entity.ChatResponse, error) {
	if m.responseToReturn != nil {
		return m.responseToReturn, nil
	}
	model := req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &entity.ChatResponse{
		ResponseText:    "Mock response from " + model,
		ModelUsed:       model,
		PromptTokens:    12,
		CandidateTokens: 24,
		TotalTokens:     36,
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (m *MockLLMService) GetAvailableModels(ctx context.Context) ([]string, error) {
	return []string{"gemini-2.5-flash", "gemini-2.5-pro", "gemini-2.0-flash"}, nil
}

type MockChatRepository struct {
	histories []*entity.ChatHistory
}

func (m *MockChatRepository) Save(ctx context.Context, history *entity.ChatHistory) error {
	m.histories = append(m.histories, history)
	return nil
}

func (m *MockChatRepository) GetHistoryByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*entity.ChatHistory, error) {
	var res []*entity.ChatHistory
	for _, h := range m.histories {
		if h.UserID != nil && *h.UserID == userID {
			res = append(res, h)
		}
	}
	return res, nil
}

type MockUserRepository struct {
	users map[uuid.UUID]*entity.User
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
	m.users[user.ID] = user
	return nil
}

type MockGoogleAuthVerifier struct{}

func (m *MockGoogleAuthVerifier) VerifyIDToken(ctx context.Context, idToken string) (*entity.GoogleClaims, error) {
	return &entity.GoogleClaims{
		GoogleID:      "google-id-12345",
		Email:         "testuser@example.com",
		EmailVerified: true,
		Name:          "Test User",
		Picture:       "https://avatar.example.com/photo.jpg",
	}, nil
}

func (m *MockGoogleAuthVerifier) ExchangeCode(ctx context.Context, authCode string) (*entity.GoogleClaims, error) {
	return &entity.GoogleClaims{
		GoogleID:      "google-id-code-123",
		Email:         "codeuser@example.com",
		EmailVerified: true,
		Name:          "Code User",
	}, nil
}

func (m *MockGoogleAuthVerifier) GetAuthURL(state string) string {
	return "https://accounts.google.com/o/oauth2/v2/auth?state=" + state
}

type MockTokenService struct {
	testUserID uuid.UUID
}

func (m *MockTokenService) GenerateToken(user *entity.User) (string, time.Time, error) {
	return "test-jwt-token-string", time.Now().Add(24 * time.Hour), nil
}

func (m *MockTokenService) ValidateToken(tokenString string) (*entity.UserClaims, error) {
	return &entity.UserClaims{
		UserID: m.testUserID,
		Email:  "testuser@example.com",
		Name:   "Test User",
	}, nil
}

func setupTestServer(t *testing.T) (*deliveryMCP.MCPServer, *MockUserRepository, *MockChatRepository, uuid.UUID) {
	testUserID := uuid.New()
	userRepo := &MockUserRepository{users: make(map[uuid.UUID]*entity.User)}
	chatRepo := &MockChatRepository{}
	llmSvc := &MockLLMService{}
	googleVerifier := &MockGoogleAuthVerifier{}
	tokenSvc := &MockTokenService{testUserID: testUserID}

	// Seed user
	user := entity.NewUser("testuser@example.com", "Test User", "https://avatar.example.com/photo.jpg", "google-id-12345")
	user.ID = testUserID
	_ = userRepo.Create(context.Background(), user)

	// Seed chat history
	_ = chatRepo.Save(context.Background(), &entity.ChatHistory{
		ID:           uuid.New(),
		UserID:       &testUserID,
		Model:        "gemini-2.5-flash",
		Prompt:       "Hello world",
		Response:     "Hello! How can I help you?",
		PromptTokens: 5,
		OutputTokens: 10,
		CreatedAt:    time.Now().UTC(),
	})

	authUC := usecase.NewAuthUseCase(userRepo, googleVerifier, tokenSvc)
	chatUC := usecase.NewChatUseCase(llmSvc, chatRepo)

	server := deliveryMCP.NewMCPServer(deliveryMCP.ServerConfig{
		AuthUseCase:    authUC,
		ChatUseCase:    chatUC,
		TokenService:   tokenSvc,
		AppName:        "gosling-mcp-test",
		AppVersion:     "1.0.0",
		AppEnv:         "test",
		AllowedOrigins: "*",
	})

	return server, userRepo, chatRepo, testUserID
}

func connectInMemory(t *testing.T, s *deliveryMCP.MCPServer) (*mcp.ClientSession, func()) {
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = s.Server().Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("failed to connect in-memory client: %v", err)
	}

	cleanup := func() {
		_ = session.Close()
		cancel()
	}

	return session, cleanup
}

// --- Tests ---

func TestMCPServer_ListTools(t *testing.T) {
	server, _, _, _ := setupTestServer(t)
	session, cleanup := connectInMemory(t, server)
	defer cleanup()

	ctx := context.Background()
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	expectedTools := map[string]bool{
		"chat":                   false,
		"get_available_models":   false,
		"get_chat_history":       false,
		"verify_google_id_token": false,
		"login_with_google_code": false,
		"get_google_auth_url":    false,
		"get_user_profile":       false,
		"health_check":           false,
	}

	for _, tool := range toolsResult.Tools {
		if _, ok := expectedTools[tool.Name]; ok {
			expectedTools[tool.Name] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("expected tool '%s' to be registered, but was missing", name)
		}
	}
}

func TestMCPServer_CallHealthCheckTool(t *testing.T) {
	server, _, _, _ := setupTestServer(t)
	session, cleanup := connectInMemory(t, server)
	defer cleanup()

	ctx := context.Background()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "health_check",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool health_check error: %v", err)
	}

	if res.IsError {
		t.Fatalf("health_check returned error result")
	}

	if len(res.Content) == 0 {
		t.Fatalf("expected content in health_check response")
	}
}

func TestMCPServer_CallChatTool(t *testing.T) {
	server, _, _, testUserID := setupTestServer(t)
	session, cleanup := connectInMemory(t, server)
	defer cleanup()

	ctx := context.Background()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "chat",
		Arguments: map[string]any{
			"prompt":     "Explain Go concurrency in 2 sentences.",
			"model":      "gemini-2.5-flash",
			"auth_token": "Bearer test-jwt-token-string",
			"user_id":    testUserID.String(),
		},
	})
	if err != nil {
		t.Fatalf("CallTool chat error: %v", err)
	}

	if res.IsError {
		t.Fatalf("chat tool returned error: %v", res.Content)
	}

	if len(res.Content) == 0 {
		t.Fatalf("expected non-empty response content from chat tool")
	}
}

func TestMCPServer_CallAuthTools(t *testing.T) {
	server, _, _, testUserID := setupTestServer(t)
	session, cleanup := connectInMemory(t, server)
	defer cleanup()

	ctx := context.Background()

	// 1. verify_google_id_token
	res1, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "verify_google_id_token",
		Arguments: map[string]any{
			"id_token": "valid-mock-token",
		},
	})
	if err != nil || res1.IsError {
		t.Fatalf("verify_google_id_token failed: %v, res: %v", err, res1)
	}

	// 2. login_with_google_code
	res2, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "login_with_google_code",
		Arguments: map[string]any{
			"code": "valid-mock-code",
		},
	})
	if err != nil || res2.IsError {
		t.Fatalf("login_with_google_code failed: %v, res: %v", err, res2)
	}

	// 3. get_google_auth_url
	res3, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_google_auth_url",
		Arguments: map[string]any{
			"state": "custom-state",
		},
	})
	if err != nil || res3.IsError {
		t.Fatalf("get_google_auth_url failed: %v, res: %v", err, res3)
	}

	// 4. get_user_profile
	res4, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_user_profile",
		Arguments: map[string]any{
			"user_id": testUserID.String(),
		},
	})
	if err != nil || res4.IsError {
		t.Fatalf("get_user_profile failed: %v, res: %v", err, res4)
	}

	// 5. get_chat_history
	res5, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_chat_history",
		Arguments: map[string]any{
			"user_id": testUserID.String(),
			"limit":   10,
		},
	})
	if err != nil || res5.IsError {
		t.Fatalf("get_chat_history failed: %v, res: %v", err, res5)
	}
}

func TestMCPServer_Prompts(t *testing.T) {
	server, _, _, _ := setupTestServer(t)
	session, cleanup := connectInMemory(t, server)
	defer cleanup()

	ctx := context.Background()

	// List prompts
	promptsRes, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatalf("ListPrompts error: %v", err)
	}

	expectedPrompts := map[string]bool{
		"gosling_chat":     false,
		"code_assistant":   false,
		"system_architect": false,
	}

	for _, p := range promptsRes.Prompts {
		if _, ok := expectedPrompts[p.Name]; ok {
			expectedPrompts[p.Name] = true
		}
	}

	for name, found := range expectedPrompts {
		if !found {
			t.Errorf("expected prompt '%s' to be registered", name)
		}
	}

	// Get prompt: gosling_chat
	promptRes, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "gosling_chat",
		Arguments: map[string]string{
			"task": "Explain microservices",
		},
	})
	if err != nil {
		t.Fatalf("GetPrompt gosling_chat error: %v", err)
	}

	if len(promptRes.Messages) == 0 {
		t.Errorf("expected prompt messages to be returned")
	}
}

func TestMCPServer_Resources(t *testing.T) {
	server, _, _, testUserID := setupTestServer(t)
	session, cleanup := connectInMemory(t, server)
	defer cleanup()

	ctx := context.Background()

	// Read static resource: gosling://models
	resModels, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "gosling://models",
	})
	if err != nil {
		t.Fatalf("ReadResource gosling://models error: %v", err)
	}
	if len(resModels.Contents) == 0 || resModels.Contents[0].Text == "" {
		t.Errorf("expected models resource content")
	}

	// Read static resource: gosling://system/health
	resHealth, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "gosling://system/health",
	})
	if err != nil {
		t.Fatalf("ReadResource gosling://system/health error: %v", err)
	}
	if len(resHealth.Contents) == 0 || resHealth.Contents[0].Text == "" {
		t.Errorf("expected health resource content")
	}

	// Read template resource: gosling://users/{user_id}/profile
	resProfile, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "gosling://users/" + testUserID.String() + "/profile",
	})
	if err != nil {
		t.Fatalf("ReadResource user profile error: %v", err)
	}
	if len(resProfile.Contents) == 0 || resProfile.Contents[0].Text == "" {
		t.Errorf("expected profile resource content")
	}
}

func TestMCPServer_HTTPTransport(t *testing.T) {
	server, _, _, _ := setupTestServer(t)
	ts := httptest.NewServer(server.HTTPHandler())
	defer ts.Close()

	// 1. Test /health endpoint
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK from /health, got %d", resp.StatusCode)
	}

	var healthData map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&healthData); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if healthData["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", healthData["status"])
	}

	// 2. Test Streamable HTTP Client Transport
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "http-client", Version: "1.0.0"}, nil)
	streamableTransport := &mcp.StreamableClientTransport{
		Endpoint: ts.URL + "/mcp",
	}

	streamableSession, err := client.Connect(ctx, streamableTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect via Streamable HTTP: %v", err)
	}
	defer streamableSession.Close()

	// Call health_check via Streamable HTTP session
	toolRes, err := streamableSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "health_check",
		Arguments: map[string]any{},
	})
	if err != nil || toolRes.IsError {
		t.Fatalf("failed calling health_check via Streamable HTTP: %v", err)
	}

	// 3. Test SSE Client Transport
	sseTransport := &mcp.SSEClientTransport{
		Endpoint: ts.URL + "/sse",
	}
	sseClient := mcp.NewClient(&mcp.Implementation{Name: "sse-client", Version: "1.0.0"}, nil)
	sseSession, err := sseClient.Connect(ctx, sseTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect via SSE: %v", err)
	}
	defer sseSession.Close()

	// Call get_available_models via SSE session
	modelsRes, err := sseSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_available_models",
		Arguments: map[string]any{},
	})
	if err != nil || modelsRes.IsError {
		t.Fatalf("failed calling get_available_models via SSE: %v", err)
	}
}
