package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *MCPServer) registerResources() {
	// 1. Static Resource: gosling://models
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "gosling://models",
		Name:        "Available LLM Models",
		Description: "List of available and supported Gemini LLM models in Gosling.",
		MIMEType:    "application/json",
	}, s.handleModelsResource)

	// 2. Static Resource: gosling://system/health
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "gosling://system/health",
		Name:        "System Health Status",
		Description: "Current operational status, version, and supported transports of Gosling MCP Server.",
		MIMEType:    "application/json",
	}, s.handleHealthResource)

	// 3. Resource Template: gosling://users/{user_id}/profile
	s.mcpServer.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "gosling://users/{user_id}/profile",
		Name:        "User Profile",
		Description: "User account profile details by user UUID.",
		MIMEType:    "application/json",
	}, s.handleUserProfileResource)

	// 4. Resource Template: gosling://users/{user_id}/history
	s.mcpServer.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "gosling://users/{user_id}/history",
		Name:        "User Chat History",
		Description: "Conversation history logs for a user by user UUID.",
		MIMEType:    "application/json",
	}, s.handleUserHistoryResource)
}

func (s *MCPServer) handleModelsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	models, err := s.cfg.ChatUseCase.GetAvailableModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve models: %w", err)
	}

	payload := map[string]any{
		"default_model": "gemini-2.5-flash",
		"models":        models,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal models resource: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

func (s *MCPServer) handleHealthResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	health := map[string]any{
		"status":      "healthy",
		"service":     s.cfg.AppName,
		"environment": s.cfg.AppEnv,
		"version":     s.cfg.AppVersion,
		"transports":  []string{"stdio", "streamable-http", "sse"},
	}

	data, err := json.MarshalIndent(health, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal health resource: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

func (s *MCPServer) handleUserProfileResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	if s.cfg.AuthUseCase == nil {
		return nil, fmt.Errorf("auth usecase is not configured")
	}

	// Extract user_id from URI: gosling://users/{user_id}/profile
	uri := req.Params.URI
	parts := strings.Split(strings.TrimPrefix(uri, "gosling://users/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		return nil, fmt.Errorf("invalid user profile URI: %s", uri)
	}

	userID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID '%s': %w", parts[0], err)
	}

	user, err := s.cfg.AuthUseCase.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user profile: %w", err)
	}

	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user profile: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}

func (s *MCPServer) handleUserHistoryResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	// Extract user_id from URI: gosling://users/{user_id}/history
	uri := req.Params.URI
	parts := strings.Split(strings.TrimPrefix(uri, "gosling://users/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		return nil, fmt.Errorf("invalid user history URI: %s", uri)
	}

	userID, err := uuid.Parse(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid user UUID '%s': %w", parts[0], err)
	}

	history, err := s.cfg.ChatUseCase.GetUserChatHistory(ctx, userID, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user history: %w", err)
	}

	payload := map[string]any{
		"user_id": userID.String(),
		"count":   len(history),
		"history": history,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user history: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		},
	}, nil
}
