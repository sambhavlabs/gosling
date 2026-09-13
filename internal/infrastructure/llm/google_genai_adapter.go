package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sourabhmandal/gosling/internal/domain/entity"
	"github.com/sourabhmandal/gosling/internal/domain/service"
	"google.golang.org/genai"
)

type googleGenAIAdapter struct {
	client       *genai.Client
	defaultModel string
}

// NewGoogleGenAIAdapter creates a new Google GenAI client adapter.
func NewGoogleGenAIAdapter(ctx context.Context, apiKey string, defaultModel string) (service.LLMService, error) {
	if apiKey == "" {
		return nil, errors.New("gemini api key is not configured")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create google genai client: %w", err)
	}

	if defaultModel == "" {
		defaultModel = "gemini-2.5-flash"
	}

	return &googleGenAIAdapter{
		client:       client,
		defaultModel: defaultModel,
	}, nil
}

func (g *googleGenAIAdapter) GenerateContent(ctx context.Context, req *entity.ChatRequest) (*entity.ChatResponse, error) {
	modelName := req.Model
	if modelName == "" {
		modelName = g.defaultModel
	}

	// Construct message contents
	var contents []*genai.Content
	if len(req.Messages) > 0 {
		for _, msg := range req.Messages {
			role := string(genai.RoleUser)
			if msg.Role == entity.RoleModel {
				role = string(genai.RoleModel)
			}
			contents = append(contents, &genai.Content{
				Role: role,
				Parts: []*genai.Part{
					{Text: msg.Content},
				},
			})
		}
	} else if req.Prompt != "" {
		contents = genai.Text(req.Prompt)
	} else {
		return nil, errors.New("empty prompt or messages in chat request")
	}

	// Build generation configuration
	config := &genai.GenerateContentConfig{}
	if req.Temperature != nil {
		config.Temperature = req.Temperature
	}
	if req.MaxOutputTokens != nil {
		config.MaxOutputTokens = *req.MaxOutputTokens
	}
	if req.SystemInstruction != "" {
		config.SystemInstruction = &genai.Content{
			Role: "system",
			Parts: []*genai.Part{
				{Text: req.SystemInstruction},
			},
		}
	}

	// Call Google GenAI SDK
	resp, err := g.client.Models.GenerateContent(ctx, modelName, contents, config)
	if err != nil {
		return nil, fmt.Errorf("google genai api call failed: %w", err)
	}

	var promptTokens, candidateTokens, totalTokens int32
	if resp.UsageMetadata != nil {
		promptTokens = resp.UsageMetadata.PromptTokenCount
		candidateTokens = resp.UsageMetadata.CandidatesTokenCount
		totalTokens = resp.UsageMetadata.TotalTokenCount
	}

	return &entity.ChatResponse{
		ResponseText:    resp.Text(),
		ModelUsed:       modelName,
		PromptTokens:    promptTokens,
		CandidateTokens: candidateTokens,
		TotalTokens:     totalTokens,
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (g *googleGenAIAdapter) GetAvailableModels(ctx context.Context) ([]string, error) {
	// Returns standard Google Gemini and reasoning models
	knownModels := []string{
		"gemini-2.5-flash",
		"gemini-2.5-pro",
		"gemini-2.0-flash",
		"gemini-2.0-flash-lite",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
	}

	return knownModels, nil
}
