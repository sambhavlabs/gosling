package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	deliveryMCP "github.com/sourabhmandal/gosling/internal/delivery/mcp"
	"github.com/sourabhmandal/gosling/internal/domain/repository"
	"github.com/sourabhmandal/gosling/internal/infrastructure/config"
	"github.com/sourabhmandal/gosling/internal/infrastructure/database"
	"github.com/sourabhmandal/gosling/internal/infrastructure/llm"
	infraRepo "github.com/sourabhmandal/gosling/internal/infrastructure/repository"
	"github.com/sourabhmandal/gosling/internal/infrastructure/security"
	"github.com/sourabhmandal/gosling/internal/usecase"
)

func main() {
	// Parse CLI flags
	stdioFlag := flag.Bool("stdio", false, "Run MCP server over JSON-RPC 2.0 stdio transport")
	transportFlag := flag.String("transport", "", "MCP transport to use: http, streamable, sse, stdio (overrides MCP_TRANSPORT env)")
	portFlag := flag.String("port", "", "Port for HTTP server (overrides PORT env)")
	flag.Parse()

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[Config Error] Failed to load configuration: %v", err)
	}

	if *portFlag != "" {
		cfg.AppPort = *portFlag
	}

	useStdio := *stdioFlag || cfg.MCPTransport == "stdio" || *transportFlag == "stdio"

	// Only print startup banner to stderr/log if not running in stdio mode (to keep stdout clean for JSON-RPC 2.0)
	if !useStdio {
		log.Println("======================================================")
		log.Println("       GOSLING - Model Context Protocol (MCP) Server  ")
		log.Println("======================================================")
		log.Printf("[Config] Environment: %s, Port: %s, Gemini Model: %s", cfg.AppEnv, cfg.AppPort, cfg.GeminiDefaultModel)
	}

	// 2. Initialize Database Connection Pool (PostgreSQL)
	var userRepo repository.UserRepository
	var chatRepo repository.ChatRepository

	db, err := database.NewPostgresDB(cfg.DatabaseDSN())
	if err != nil {
		if !useStdio {
			log.Printf("[Database Warning] PostgreSQL connection not established (%v). Running without database persistence.", err)
		}
	} else {
		defer func() {
			if err := db.Close(); err != nil && !useStdio {
				log.Printf("[Database] Error closing DB pool: %v", err)
			}
		}()

		if err := database.RunMigrations(db); err != nil {
			if !useStdio {
				log.Printf("[Migration Warning] Failed to run migrations: %v", err)
			}
		} else if !useStdio {
			log.Println("[Database] Connected and migrations applied successfully.")
		}

		userRepo = infraRepo.NewUserRepositoryPostgres(db)
		chatRepo = infraRepo.NewChatRepositoryPostgres(db)
	}

	// 3. Initialize Security Adapters (JWT & Google Auth)
	jwtService := security.NewJWTService(cfg.JWTSecret, cfg.JWTExpiryHours)
	googleAuthService := security.NewGoogleAuthService(
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleRedirectURL,
	)

	// 4. Initialize Google GenAI LLM Adapter
	ctx := context.Background()
	llmService, err := llm.NewGoogleGenAIAdapter(ctx, cfg.GeminiAPIKey, cfg.GeminiDefaultModel)
	if err != nil && !useStdio {
		log.Printf("[LLM Warning] Initialized Google GenAI adapter notice: %v", err)
	} else if !useStdio {
		log.Printf("[LLM] Google GenAI adapter active with model: %s", cfg.GeminiDefaultModel)
	}

	// 5. Initialize Application Use Cases (Dependency Injection)
	authUseCase := usecase.NewAuthUseCase(userRepo, googleAuthService, jwtService)
	chatUseCase := usecase.NewChatUseCase(llmService, chatRepo)

	// 6. Build MCP Server
	mcpServer := deliveryMCP.NewMCPServer(deliveryMCP.ServerConfig{
		AuthUseCase:    authUseCase,
		ChatUseCase:    chatUseCase,
		TokenService:   jwtService,
		AppName:        cfg.AppName,
		AppVersion:     cfg.AppVersion,
		AppEnv:         cfg.AppEnv,
		AllowedOrigins: cfg.AllowedOrigins,
	})

	// 7. Start Server on Selected Transport
	if useStdio {
		// Run MCP over standard input/output with JSON-RPC 2.0 messages
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			cancel()
		}()

		if err := mcpServer.RunStdio(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Fatalf("[MCP Stdio] Server error: %v", err)
		}
		return
	}

	// HTTP Mode: Supports Streamable HTTP (/mcp), SSE (/sse), and Health (/health)
	httpHandler := mcpServer.HTTPHandler()
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.AppPort),
		Handler:      httpHandler,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  180 * time.Second,
	}

	go func() {
		log.Printf("[MCP Server] Listening on http://0.0.0.0:%s", cfg.AppPort)
		log.Printf("[MCP Endpoints] Streamable HTTP: http://0.0.0.0:%s/mcp | SSE: http://0.0.0.0:%s/sse | Health: http://0.0.0.0:%s/health", cfg.AppPort, cfg.AppPort, cfg.AppPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[MCP HTTP Server] Error: %v", err)
		}
	}()

	// Graceful shutdown listener
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Shutdown] Shutting down MCP server gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[Shutdown] Server forced to shutdown: %v", err)
	}

	log.Println("[Shutdown] Server exited cleanly.")
}
