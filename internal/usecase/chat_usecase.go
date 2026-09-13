package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sourabhmandal/gosling/internal/domain"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
	"github.com/sourabhmandal/gosling/internal/domain/repository"
	"github.com/sourabhmandal/gosling/internal/domain/service"
)

// ChatUseCase defines the application operations for interacting with LLMs.
type ChatUseCase interface {
	SendMessage(ctx context.Context, userID *uuid.UUID, req *entity.ChatRequest) (*entity.ChatResponse, error)
	GetAvailableModels(ctx context.Context) ([]string, error)
	GetUserChatHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*entity.ChatHistory, error)
}

type chatUseCase struct {
	llmService service.LLMService
	chatRepo   repository.ChatRepository
}

// NewChatUseCase creates an instance of ChatUseCase with injected dependencies.
func NewChatUseCase(
	llmService service.LLMService,
	chatRepo repository.ChatRepository,
) ChatUseCase {
	return &chatUseCase{
		llmService: llmService,
		chatRepo:   chatRepo,
	}
}

func (c *chatUseCase) SendMessage(ctx context.Context, userID *uuid.UUID, req *entity.ChatRequest) (*entity.ChatResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: chat request cannot be nil", domain.ErrInvalidInput)
	}

	// Validate input
	if strings.TrimSpace(req.Prompt) == "" && len(req.Messages) == 0 {
		return nil, fmt.Errorf("%w: prompt or messages must be provided", domain.ErrInvalidInput)
	}

	if c.llmService == nil {
		return nil, fmt.Errorf("%w: LLM service is not configured (GEMINI_API_KEY is missing)", domain.ErrLLMExecutionFailed)
	}

	// Call LLM service adapter
	resp, err := c.llmService.GenerateContent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrLLMExecutionFailed, err)
	}

	// Optionally persist conversation history if repository is available
	if c.chatRepo != nil {
		promptSummary := req.Prompt
		if promptSummary == "" && len(req.Messages) > 0 {
			promptSummary = req.Messages[len(req.Messages)-1].Content
		}

		history := &entity.ChatHistory{
			ID:           uuid.New(),
			UserID:       userID,
			Model:        resp.ModelUsed,
			Prompt:       promptSummary,
			Response:     resp.ResponseText,
			PromptTokens: resp.PromptTokens,
			OutputTokens: resp.CandidateTokens,
			CreatedAt:    time.Now().UTC(),
		}

		// Save asynchronously or safely ignore error to not fail generation
		go func(h *entity.ChatHistory) {
			_ = c.chatRepo.Save(context.Background(), h)
		}(history)
	}

	return resp, nil
}

func (c *chatUseCase) GetAvailableModels(ctx context.Context) ([]string, error) {
	if c.llmService == nil {
		return []string{
			"gemini-2.5-flash",
			"gemini-2.5-pro",
			"gemini-2.0-flash",
			"gemini-2.0-flash-lite",
			"gemini-1.5-pro",
			"gemini-1.5-flash",
		}, nil
	}
	return c.llmService.GetAvailableModels(ctx)
}

func (c *chatUseCase) GetUserChatHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*entity.ChatHistory, error) {
	if c.chatRepo == nil {
		return []*entity.ChatHistory{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	return c.chatRepo.GetHistoryByUserID(ctx, userID, limit)
}
