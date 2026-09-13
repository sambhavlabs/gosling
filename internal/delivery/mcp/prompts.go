package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *MCPServer) registerPrompts() {
	// 1. Prompt: gosling_chat
	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "gosling_chat",
		Description: "General conversational and task completion prompt template for Gosling AI.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "task",
				Description: "The core task, question, or request for the model to perform",
				Required:    true,
			},
			{
				Name:        "context",
				Description: "Additional background context or constraints",
				Required:    false,
			},
			{
				Name:        "system_instruction",
				Description: "Optional persona or system instructions",
				Required:    false,
			},
		},
	}, s.handleGoslingChatPrompt)

	// 2. Prompt: code_assistant
	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "code_assistant",
		Description: "Specialized software development and code generation prompt template.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "task",
				Description: "The coding task, feature to implement, or bug to fix",
				Required:    true,
			},
			{
				Name:        "language",
				Description: "Programming language (e.g. Go, Python, TypeScript)",
				Required:    false,
			},
			{
				Name:        "framework",
				Description: "Framework, libraries, or architecture pattern (e.g. DDD, Gin, React)",
				Required:    false,
			},
		},
	}, s.handleCodeAssistantPrompt)

	// 3. Prompt: system_architect
	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "system_architect",
		Description: "Domain Driven Design and distributed systems architectural review and planning prompt template.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "requirements",
				Description: "System functional and non-functional requirements",
				Required:    true,
			},
			{
				Name:        "patterns",
				Description: "Target architectural patterns (e.g. Onion Architecture, Event-Driven, Microservices, CQRS)",
				Required:    false,
			},
		},
	}, s.handleSystemArchitectPrompt)
}

func (s *MCPServer) handleGoslingChatPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	task := req.Params.Arguments["task"]
	if task == "" {
		return nil, fmt.Errorf("argument 'task' is required")
	}

	contextInfo := req.Params.Arguments["context"]
	systemInstruction := req.Params.Arguments["system_instruction"]

	var messages []*mcp.PromptMessage

	if systemInstruction != "" {
		messages = append(messages, &mcp.PromptMessage{
			Role: mcp.Role("user"),
			Content: &mcp.TextContent{
				Text: fmt.Sprintf("[SYSTEM INSTRUCTION]: %s", systemInstruction),
			},
		})
	}

	userPrompt := task
	if contextInfo != "" {
		userPrompt = fmt.Sprintf("Context:\n%s\n\nTask:\n%s", contextInfo, task)
	}

	messages = append(messages, &mcp.PromptMessage{
		Role: mcp.Role("user"),
		Content: &mcp.TextContent{
			Text: userPrompt,
		},
	})

	return &mcp.GetPromptResult{
		Description: "Gosling AI Chat Prompt",
		Messages:    messages,
	}, nil
}

func (s *MCPServer) handleCodeAssistantPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	task := req.Params.Arguments["task"]
	if task == "" {
		return nil, fmt.Errorf("argument 'task' is required")
	}

	lang := req.Params.Arguments["language"]
	if lang == "" {
		lang = "Go"
	}

	framework := req.Params.Arguments["framework"]

	promptText := fmt.Sprintf("You are an expert %s software engineer. Please implement the following task with high quality, idiomatic patterns, clear tests, and comprehensive error handling.\n", lang)
	if framework != "" {
		promptText += fmt.Sprintf("Framework/Patterns: %s\n", framework)
	}
	promptText += fmt.Sprintf("\nTask:\n%s", task)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("%s Code Assistant Prompt", lang),
		Messages: []*mcp.PromptMessage{
			{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: promptText,
				},
			},
		},
	}, nil
}

func (s *MCPServer) handleSystemArchitectPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	reqs := req.Params.Arguments["requirements"]
	if reqs == "" {
		return nil, fmt.Errorf("argument 'requirements' is required")
	}

	patterns := req.Params.Arguments["patterns"]
	if patterns == "" {
		patterns = "Domain-Driven Design (DDD), Clean/Onion Architecture, Ports & Adapters"
	}

	promptText := fmt.Sprintf(`You are a Principal Software Architect specializing in %s.
Design a robust, scalable system architecture based on the following requirements.
Include domain entity models, bounded contexts, ports (interfaces), infrastructure adapters, and data flow.

Requirements:
%s`, patterns, reqs)

	return &mcp.GetPromptResult{
		Description: "Domain Driven Design & System Architecture Prompt",
		Messages: []*mcp.PromptMessage{
			{
				Role: mcp.Role("user"),
				Content: &mcp.TextContent{
					Text: promptText,
				},
			},
		},
	}, nil
}
