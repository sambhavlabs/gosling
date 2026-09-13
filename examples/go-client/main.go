package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	transportFlag := flag.String("transport", "streamable", "MCP transport: streamable (HTTP POST/GET), sse (Server-Sent Events), or stdio (JSON-RPC 2.0 subprocess)")
	urlFlag := flag.String("url", "", "MCP Server URL (default: http://localhost:8080/mcp for streamable, http://localhost:8080/sse for sse)")
	cmdFlag := flag.String("cmd", "", "Command to run for stdio transport (e.g. 'go run ./cmd/server/main.go --stdio')")
	promptFlag := flag.String("prompt", "Explain Domain-Driven Design in 3 concise bullet points.", "Prompt to send to the chat tool")
	modelFlag := flag.String("model", "gemini-2.5-flash", "LLM model to use (e.g. gemini-2.5-flash, gemini-2.5-pro)")
	interactiveFlag := flag.Bool("interactive", false, "Run in interactive REPL chat mode with Gosling MCP Server")
	flag.Parse()

	transportType := strings.ToLower(strings.TrimSpace(*transportFlag))
	endpointURL := *urlFlag
	if endpointURL == "" {
		if transportType == "sse" {
			endpointURL = "http://localhost:8080/sse"
		} else {
			endpointURL = "http://localhost:8080/mcp"
		}
	}

	fmt.Println("==================================================================")
	fmt.Println("       GOSLING - MCP Client Example (Go SDK)                     ")
	fmt.Println("==================================================================")
	fmt.Printf("[Client] Transport Mode: %s\n", transportType)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n[Client] Interrupt received, closing connection...")
		cancel()
	}()

	// 1. Configure the MCP Transport
	var transport mcp.Transport
	switch transportType {
	case "streamable", "http":
		fmt.Printf("[Client] Connecting to Streamable HTTP endpoint: %s\n", endpointURL)
		transport = &mcp.StreamableClientTransport{
			Endpoint: endpointURL,
		}
	case "sse":
		fmt.Printf("[Client] Connecting to SSE endpoint: %s\n", endpointURL)
		transport = &mcp.SSEClientTransport{
			Endpoint: endpointURL,
		}
	case "stdio":
		serverCmd := *cmdFlag
		if serverCmd == "" {
			serverCmd = "go run ./cmd/server/main.go --stdio"
		}
		parts := strings.Fields(serverCmd)
		if len(parts) == 0 {
			log.Fatalf("[Error] Invalid stdio command: %s", serverCmd)
		}
		fmt.Printf("[Client] Spawning subprocess for JSON-RPC 2.0 stdio: %s\n", serverCmd)
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Stderr = os.Stderr // pipe server logs to client stderr
		transport = &mcp.CommandTransport{
			Command: cmd,
		}
	default:
		log.Fatalf("[Error] Unsupported transport: %s. Use 'streamable', 'sse', or 'stdio'", transportType)
	}

	// 2. Initialize MCP Client
	clientImpl := &mcp.Implementation{
		Name:    "gosling-go-client",
		Version: "1.0.0",
	}
	client := mcp.NewClient(clientImpl, nil)

	fmt.Println("[Client] Initializing connection and negotiating capabilities...")
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("[Error] Failed to connect to MCP server: %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			log.Printf("[Warning] Error closing session: %v", err)
		}
	}()

	initResult := session.InitializeResult()
	if initResult != nil {
		fmt.Printf("[Server Info] Connected to: %s (v%s)\n", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
		if initResult.Instructions != "" {
			fmt.Printf("[Server Instructions] %s\n", initResult.Instructions)
		}
	}

	// If interactive mode is chosen, enter chat loop
	if *interactiveFlag {
		runInteractiveLoop(ctx, session, *modelFlag)
		return
	}

	// Otherwise, run automated full demonstration
	runAutomatedDemo(ctx, session, *promptFlag, *modelFlag)
}

func runAutomatedDemo(ctx context.Context, session *mcp.ClientSession, prompt string, model string) {
	fmt.Println("\n------------------------------------------------------------------")
	fmt.Println(" 1. LISTING MCP TOOLS")
	fmt.Println("------------------------------------------------------------------")
	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("[Error] Failed to list tools: %v", err)
	}
	for i, t := range toolsRes.Tools {
		fmt.Printf(" [%d] Tool: %-24s - %s\n", i+1, t.Name, t.Description)
	}

	fmt.Println("\n------------------------------------------------------------------")
	fmt.Println(" 2. CALLING TOOL: health_check")
	fmt.Println("------------------------------------------------------------------")
	healthRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "health_check",
		Arguments: map[string]any{},
	})
	if err != nil {
		log.Fatalf("[Error] health_check call failed: %v", err)
	}
	printToolResult(healthRes)

	fmt.Println("\n------------------------------------------------------------------")
	fmt.Println(" 3. CALLING TOOL: get_available_models")
	fmt.Println("------------------------------------------------------------------")
	modelsRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_available_models",
		Arguments: map[string]any{},
	})
	if err != nil {
		log.Fatalf("[Error] get_available_models call failed: %v", err)
	}
	printToolResult(modelsRes)

	fmt.Println("\n------------------------------------------------------------------")
	fmt.Printf(" 4. CALLING TOOL: chat (Model: %s)\n", model)
	fmt.Println("------------------------------------------------------------------")
	fmt.Printf("[Prompt] %s\n\n", prompt)
	chatRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "chat",
		Arguments: map[string]any{
			"model":       model,
			"prompt":      prompt,
			"temperature": 0.7,
		},
	})
	if err != nil {
		log.Printf("[Error] chat call failed: %v", err)
	} else {
		printToolResult(chatRes)
	}

	fmt.Println("\n------------------------------------------------------------------")
	fmt.Println(" 5. LISTING MCP PROMPTS & RETRIEVING A PROMPT TEMPLATE")
	fmt.Println("------------------------------------------------------------------")
	promptsRes, err := session.ListPrompts(ctx, nil)
	if err != nil {
		log.Printf("[Warning] Failed to list prompts: %v", err)
	} else {
		for i, p := range promptsRes.Prompts {
			fmt.Printf(" [%d] Prompt: %-20s - %s\n", i+1, p.Name, p.Description)
		}

		// Retrieve prompt template: gosling_chat
		pRes, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
			Name: "gosling_chat",
			Arguments: map[string]string{
				"task":    "Explain the Onion Architecture pattern",
				"context": "Gosling MCP Server Implementation",
			},
		})
		if err != nil {
			log.Printf("[Warning] GetPrompt failed: %v", err)
		} else {
			fmt.Printf("\n[Prompt Content Rendered by Server]:\n")
			for _, m := range pRes.Messages {
				if tc, ok := m.Content.(*mcp.TextContent); ok {
					fmt.Printf("- Role: %s\n  Text: %s\n", m.Role, tc.Text)
				}
			}
		}
	}

	fmt.Println("\n------------------------------------------------------------------")
	fmt.Println(" 6. LISTING & READING MCP RESOURCES")
	fmt.Println("------------------------------------------------------------------")
	resList, err := session.ListResources(ctx, nil)
	if err != nil {
		log.Printf("[Warning] ListResources error: %v", err)
	} else {
		for i, r := range resList.Resources {
			fmt.Printf(" [%d] Resource: %-25s (%s) - %s\n", i+1, r.URI, r.MIMEType, r.Name)
		}

		// Read resource: gosling://models
		modelsContent, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
			URI: "gosling://models",
		})
		if err != nil {
			log.Printf("[Warning] ReadResource error: %v", err)
		} else if len(modelsContent.Contents) > 0 {
			fmt.Printf("\n[Resource Content: %s]:\n%s\n", modelsContent.Contents[0].URI, modelsContent.Contents[0].Text)
		}
	}

	fmt.Println("\n==================================================================")
	fmt.Println("       Gosling MCP Client Demonstration Completed Successfully    ")
	fmt.Println("==================================================================")
}

func runInteractiveLoop(ctx context.Context, session *mcp.ClientSession, defaultModel string) {
	fmt.Println("\n------------------------------------------------------------------")
	fmt.Println(" INTERACTIVE CHAT MODE (type 'exit' or 'quit' to stop)")
	fmt.Println("------------------------------------------------------------------")

	scanner := bufio.NewScanner(os.Stdin)
	var conversationHistory []map[string]string

	for {
		fmt.Print("\nYou > ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			fmt.Println("Goodbye!")
			break
		}

		// Append user message
		conversationHistory = append(conversationHistory, map[string]string{
			"role":    "user",
			"content": line,
		})

		// Call chat tool
		callParams := &mcp.CallToolParams{
			Name: "chat",
			Arguments: map[string]any{
				"model":    defaultModel,
				"messages": conversationHistory,
			},
		}

		fmt.Println("\n[Gosling AI is thinking...]")
		callRes, err := session.CallTool(ctx, callParams)
		if err != nil {
			fmt.Printf("[Error] Tool call failed: %v\n", err)
			continue
		}

		if callRes.IsError {
			fmt.Printf("[Model Error] %v\n", callRes.Content)
			continue
		}

		// Parse structured content or text content
		var responseText string
		if len(callRes.Content) > 0 {
			if tc, ok := callRes.Content[0].(*mcp.TextContent); ok {
				var parsed struct {
					ResponseText string `json:"response_text"`
				}
				if err := json.Unmarshal([]byte(tc.Text), &parsed); err == nil && parsed.ResponseText != "" {
					responseText = parsed.ResponseText
				} else {
					responseText = tc.Text
				}
			}
		}

		fmt.Printf("\nGosling > %s\n", responseText)

		// Append model response to conversation history
		conversationHistory = append(conversationHistory, map[string]string{
			"role":    "model",
			"content": responseText,
		})
	}
}

func printToolResult(res *mcp.CallToolResult) {
	if res.IsError {
		fmt.Printf("[Error Result]: %v\n", res.Content)
		return
	}

	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			var formatted map[string]any
			if err := json.Unmarshal([]byte(tc.Text), &formatted); err == nil {
				pretty, _ := json.MarshalIndent(formatted, "", "  ")
				fmt.Println(string(pretty))
			} else {
				fmt.Println(tc.Text)
			}
		}
	}
}
