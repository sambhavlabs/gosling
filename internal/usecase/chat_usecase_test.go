package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
	"github.com/sourabhmandal/gosling/internal/usecase"
)

// MockLLMService implements service.LLMService
type MockLLMService struct {
	responseToReturn *entity.ChatResponse
	errToReturn      error
}

func (m *MockLLMService) GenerateContent(ctx context.Context, req *entity.ChatRequest) (*entity.ChatResponse, error) {
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	if m.responseToReturn != nil {
		return m.responseToReturn, nil
	}
	model := req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &entity.ChatResponse{
		ResponseText:    "Hello! How can I assist you today?",
		ModelUsed:       model,
		PromptTokens:    10,
		CandidateTokens: 15,
		TotalTokens:     25,
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (m *MockLLMService) GetAvailableModels(ctx context.Context) ([]string, error) {
	return []string{"gemini-2.5-flash", "gemini-2.5-pro"}, nil
}

// MockChatRepository implements repository.ChatRepository
type MockChatRepository struct {
	histories []*entity.ChatHistory
}

func (m *MockChatRepository) Save(ctx context.Context, history *entity.ChatHistory) error {
	m.histories = append(m.histories, history)
	return nil
}

func (m *MockChatRepository) GetHistoryByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*entity.ChatHistory, error) {
	var result []*entity.ChatHistory
	for _, h := range m.histories {
		if h.UserID != nil && *h.UserID == userID {
			result = append(result, h)
		}
	}
	return result, nil
}

func TestChatUseCase_SendMessage(t *testing.T) {
	llmSvc := &MockLLMService{}
	chatRepo := &MockChatRepository{}
	chatUC := usecase.NewChatUseCase(llmSvc, chatRepo)

	req := &entity.ChatRequest{
		Prompt: "Explain Domain Driven Design in simple terms",
		Model:  "gemini-2.5-flash",
	}

	userID := uuid.New()
	resp, err := chatUC.SendMessage(context.Background(), &userID, req)
	if err != nil {
		t.Fatalf("unexpected error calling SendMessage: %v", err)
	}

	if resp.ResponseText == "" {
		t.Errorf("expected non-empty response text")
	}

	if resp.ModelUsed != "gemini-2.5-flash" {
		t.Errorf("expected model to be gemini-2.5-flash, got %s", resp.ModelUsed)
	}

	if resp.TotalTokens != 25 {
		t.Errorf("expected 25 total tokens, got %d", resp.TotalTokens)
	}
}

func TestChatUseCase_EmptyPromptError(t *testing.T) {
	llmSvc := &MockLLMService{}
	chatUC := usecase.NewChatUseCase(llmSvc, nil)

	req := &entity.ChatRequest{
		Prompt: "",
	}

	_, err := chatUC.SendMessage(context.Background(), nil, req)
	if err == nil {
		t.Fatal("expected error when prompt and messages are empty")
	}
}
