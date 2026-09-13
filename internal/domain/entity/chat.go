package entity

import (
	"time"

	"github.com/google/uuid"
)

// MessageRole represents the sender role in a conversation.
type MessageRole string

const (
	RoleUser   MessageRole = "user"
	RoleModel  MessageRole = "model"
	RoleSystem MessageRole = "system"
)

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

// ChatRequest represents the domain payload to request LLM generation.
type ChatRequest struct {
	Model             string        `json:"model,omitempty"`              // Target LLM (e.g. "gemini-2.5-flash", "gemini-2.5-pro", etc.)
	Prompt            string        `json:"prompt"`                       // Primary prompt (if messages array is empty)
	Messages          []ChatMessage `json:"messages,omitempty"`           // Multi-turn conversation messages
	SystemInstruction string        `json:"system_instruction,omitempty"` // Optional system prompt
	Temperature       *float32      `json:"temperature,omitempty"`        // Randomness parameter (0.0 to 2.0)
	MaxOutputTokens   *int32        `json:"max_output_tokens,omitempty"`  // Maximum tokens to generate
}

// ChatResponse represents the LLM response output.
type ChatResponse struct {
	ResponseText    string    `json:"response_text"`
	ModelUsed       string    `json:"model_used"`
	PromptTokens    int32     `json:"prompt_tokens"`
	CandidateTokens int32     `json:"candidate_tokens"`
	TotalTokens     int32     `json:"total_tokens"`
	CreatedAt       time.Time `json:"created_at"`
}

// ChatHistory represents a saved conversation interaction in persistence.
type ChatHistory struct {
	ID           uuid.UUID  `json:"id"`
	UserID       *uuid.UUID `json:"user_id,omitempty"` // nullable for unauthenticated chat
	Model        string     `json:"model"`
	Prompt       string     `json:"prompt"`
	Response     string     `json:"response"`
	PromptTokens int32      `json:"prompt_tokens"`
	OutputTokens int32      `json:"output_tokens"`
	CreatedAt    time.Time  `json:"created_at"`
}
