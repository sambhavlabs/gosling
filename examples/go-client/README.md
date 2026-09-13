# Gosling MCP Go Client

A sample Golang MCP client application demonstrating how to interact with the **Gosling MCP Server** using the official [`github.com/modelcontextprotocol/go-sdk/mcp`](https://github.com/modelcontextprotocol/go-sdk) library.

Supports all standard MCP transports:
- **Streamable HTTP** (`http://localhost:8080/mcp`)
- **Server-Sent Events (SSE)** (`http://localhost:8080/sse`)
- **JSON-RPC 2.0 Stdio Subprocess** (`go run ./cmd/server/main.go --stdio`)

---

## 1. Quick Start

Ensure the Gosling MCP server is running (or test directly via stdio):

### Option A: Connect via Streamable HTTP (Default)
```bash
go run ./examples/go-client/main.go -transport=streamable -url=http://localhost:8080/mcp
```

### Option B: Connect via Server-Sent Events (SSE)
```bash
go run ./examples/go-client/main.go -transport=sse -url=http://localhost:8080/sse
```

### Option C: Connect via Stdio (Direct JSON-RPC 2.0 Subprocess)
```bash
go run ./examples/go-client/main.go -transport=stdio -cmd="go run ./cmd/server/main.go --stdio"
```

---

## 2. Interactive REPL Chat Mode

You can chat interactively with Gosling via the MCP protocol:

```bash
go run ./examples/go-client/main.go -transport=streamable -interactive
```

---

## 3. Custom Prompts and Models

You can pass custom prompts and target Gemini models directly:

```bash
go run ./examples/go-client/main.go \
  -transport=streamable \
  -model=gemini-2.5-pro \
  -prompt="Explain Clean Architecture and Dependency Inversion in Go."
```

---

## 4. Features Demonstrated

1. **Protocol Handshake & Capability Negotiation**: Connects using `mcp.NewClient` and inspects `InitializeResult`.
2. **Tool Discovery**: Queries all registered server tools via `session.ListTools`.
3. **Health Checking**: Calls the `health_check` MCP tool.
4. **Model Discovery**: Calls `get_available_models` to inspect available Gemini LLMs.
5. **AI Chat Execution**: Invokes the `chat` tool with prompt parameters.
6. **Prompt Templates**: Lists prompts and retrieves rendered templates via `session.GetPrompt`.
7. **Resource Reading**: Queries and reads MCP resources (`gosling://models`) via `session.ReadResource`.
8. **Graceful Teardown**: Cleanly closes client transport and handles OS termination signals.
