package service

import (
	"context"

	"github.com/sourabhmandal/gosling/internal/domain/entity"
)

// LLMService defines the port for communicating with Large Language Models via Google GenAI SDK.
type LLMService interface {
	GenerateContent(ctx context.Context, req *entity.ChatRequest) (*entity.ChatResponse, error)
	GetAvailableModels(ctx context.Context) ([]string, error)
}
