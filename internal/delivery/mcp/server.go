package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sourabhmandal/gosling/internal/domain/service"
	"github.com/sourabhmandal/gosling/internal/usecase"
)

// ServerConfig holds the dependencies and configuration required by the MCP Server.
type ServerConfig struct {
	AuthUseCase    usecase.AuthUseCase
	ChatUseCase    usecase.ChatUseCase
	TokenService   service.TokenService
	AppName        string
	AppVersion     string
	AppEnv         string
	AllowedOrigins string
}

// MCPServer encapsulates the MCP server instance, tools, prompts, resources, and transports.
type MCPServer struct {
	mcpServer *mcp.Server
	cfg       ServerConfig
}

// NewMCPServer constructs and initializes a new Gosling MCP server with all tools, prompts, and resources.
func NewMCPServer(cfg ServerConfig) *MCPServer {
	if cfg.AppName == "" {
		cfg.AppName = "gosling-mcp-server"
	}
	if cfg.AppVersion == "" {
		cfg.AppVersion = "1.0.0"
	}

	impl := &mcp.Implementation{
		Name:    cfg.AppName,
		Version: cfg.AppVersion,
	}

	serverOpts := &mcp.ServerOptions{
		Instructions: "Gosling MCP Server provides AI chat capabilities powered by Google Gemini SDK, multi-turn LLM tools, authentication, and chat history persistence.",
	}

	rawServer := mcp.NewServer(impl, serverOpts)

	server := &MCPServer{
		mcpServer: rawServer,
		cfg:       cfg,
	}

	// Register all MCP capabilities
	server.registerTools()
	server.registerPrompts()
	server.registerResources()

	return server
}

// Server returns the underlying MCP SDK server instance.
func (s *MCPServer) Server() *mcp.Server {
	return s.mcpServer
}

// Config returns the server configuration.
func (s *MCPServer) Config() ServerConfig {
	return s.cfg
}

// RunStdio starts the MCP server over standard input and output (JSON-RPC 2.0).
func (s *MCPServer) RunStdio(ctx context.Context) error {
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}
