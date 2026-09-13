.PHONY: all build build-client run run-stdio test clean docker-up docker-down docker-logs docker-restart lint client client-sse client-stdio

# Binary names
SERVER_BINARY=gosling-mcp-server
CLIENT_BINARY=gosling-mcp-client

all: build build-client

# Run MCP Server locally (Streamable HTTP + SSE)
run:
	go run cmd/server/main.go

# Run MCP Server in JSON-RPC 2.0 Stdio mode
run-stdio:
	go run cmd/server/main.go --stdio

# Build binaries
build:
	go build -o bin/$(SERVER_BINARY) cmd/server/main.go

build-client:
	go build -o bin/$(CLIENT_BINARY) examples/go-client/main.go

# Run Example MCP Go Client (Streamable HTTP)
client:
	go run examples/go-client/main.go -transport=streamable -url=http://localhost:8080/mcp

# Run Example MCP Go Client (SSE)
client-sse:
	go run examples/go-client/main.go -transport=sse -url=http://localhost:8080/sse

# Run Example MCP Go Client (Stdio)
client-stdio:
	go run examples/go-client/main.go -transport=stdio -cmd="./bin/$(SERVER_BINARY) --stdio"

# Run tests
test:
	go test -v -race ./...

# Run clean
clean:
	rm -rf bin/

# Docker commands
docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-restart:
	docker compose restart
