package mcp

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HTTPHandler returns an http.Handler that routes MCP transports (Streamable HTTP, SSE) and health checks.
func (s *MCPServer) HTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// 1. Streamable HTTP Handler
	streamableHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server {
			return s.mcpServer
		},
		&mcp.StreamableHTTPOptions{
			DisableLocalhostProtection: true,
		},
	)

	// 2. Server-Sent Events (SSE) Handler
	sseHandler := mcp.NewSSEHandler(
		func(*http.Request) *mcp.Server {
			return s.mcpServer
		},
		&mcp.SSEOptions{
			DisableLocalhostProtection: true,
		},
	)

	// Register specific endpoints
	mux.Handle("/mcp", streamableHandler)
	mux.Handle("/mcp/", streamableHandler)
	mux.Handle("/sse", sseHandler)
	mux.Handle("/sse/", sseHandler)

	// Health check endpoint for containers / load balancers
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "healthy",
			"service":     s.cfg.AppName,
			"version":     s.cfg.AppVersion,
			"environment": s.cfg.AppEnv,
			"transports":  []string{"streamable-http", "sse", "stdio"},
			"endpoints": map[string]string{
				"streamable_http": "/mcp",
				"sse":             "/sse",
				"health":          "/health",
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Root handler: delegate based on Accept header or default to streamable
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			accept := r.Header.Get("Accept")
			if strings.Contains(accept, "text/event-stream") {
				sseHandler.ServeHTTP(w, r)
				return
			}
			streamableHandler.ServeHTTP(w, r)
			return
		}
		// Default 404 for unknown paths
		http.NotFound(w, r)
	})

	// Apply CORS and Logging Middleware
	return s.withLogging(s.withCORS(mux))
}

func (s *MCPServer) withCORS(next http.Handler) http.Handler {
	allowedOrigins := s.cfg.AllowedOrigins
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins == "*" || strings.Contains(allowedOrigins, origin) {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Mcp-Session-Id, Last-Event-ID, Cache-Control")
		w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id, Content-Length")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *MCPServer) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriterInterceptor{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("[MCP HTTP] %s %s | Status: %d | Latency: %v | Remote: %s",
			r.Method,
			r.URL.Path,
			rw.statusCode,
			duration,
			r.RemoteAddr,
		)
	})
}

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterInterceptor) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterInterceptor) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
