package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const ProtocolVersion = "2025-06-18"

// Basic JSON-RPC types
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP Types
type ClientCapabilities struct {
	Roots    *RootsCapability `json:"roots,omitempty"`
	Sampling *struct{}        `json:"sampling,omitempty"`
}

type RootsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ServerCapabilities struct {
	Logging   *struct{} `json:"logging,omitempty"`
	Prompts   *struct{} `json:"prompts,omitempty"`
	Resources *struct{} `json:"resources,omitempty"`
	Tools     *struct{} `json:"tools,omitempty"`
}

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Vendor  string `json:"vendor,omitempty"`
}

// Initialize request/response
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
}

// Tools
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// MCP Client
type MCPClient struct {
	baseURL    string
	httpClient *http.Client
	requestID  int
}

func NewMCPClient(baseURL string) *MCPClient {
	return &MCPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		requestID: 1,
	}
}

func (c *MCPClient) sendRequest(method string, params interface{}) (*JSONRPCResponse, error) {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.requestID,
		Method:  method,
		Params:  params,
	}
	c.requestID++

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	fmt.Printf("Sending request: %s\n", string(requestBody))

	// Create a proper HTTP request with headers
	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add proper headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	req.Header.Set("User-Agent", "go-mcp-client/1.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body for better error reporting
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s - Response: %s", resp.StatusCode, resp.Status, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Received response: %s\n", string(responseBody))

	var response JSONRPCResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (c *MCPClient) Initialize() error {
	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ClientCapabilities{
			Roots: &RootsCapability{
				ListChanged: true,
			},
			Sampling: &struct{}{},
		},
		ClientInfo: Implementation{
			Name:    "go-mcp-client",
			Version: "1.0.0",
			Vendor:  "example",
		},
	}

	response, err := c.sendRequest("initialize", params)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("initialize error: %s", response.Error.Message)
	}

	var result InitializeResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}

	fmt.Printf("Connected to: %s v%s\n", result.ServerInfo.Name, result.ServerInfo.Version)
	fmt.Printf("Protocol version: %s\n", result.ProtocolVersion)

	// Check capabilities
	if result.Capabilities.Tools != nil {
		fmt.Println("✓ Server supports tools")
	}
	if result.Capabilities.Resources != nil {
		fmt.Println("✓ Server supports resources")
	}
	if result.Capabilities.Prompts != nil {
		fmt.Println("✓ Server supports prompts")
	}

	return nil
}

func (c *MCPClient) ListTools() ([]Tool, error) {
	response, err := c.sendRequest("tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("list tools request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("list tools error: %s", response.Error.Message)
	}

	var result ListToolsResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list tools result: %w", err)
	}

	return result.Tools, nil
}

func (c *MCPClient) CallTool(name string, arguments map[string]interface{}) ([]byte, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	}

	response, err := c.sendRequest("tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("call tool request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("call tool error: %s", response.Error.Message)
	}

	return response.Result, nil
}

func main() {
	// Change this URL to your MCP server endpoint
	serverURL := "http://localhost:3000/mcp"

	fmt.Println("=== MCP Client - List Tools Example ===")

	// Create client
	client := NewMCPClient(serverURL)

	// Initialize connection
	fmt.Println("\n1. Initializing connection...")
	if err := client.Initialize(); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// List tools
	fmt.Println("\n2. Listing available tools...")
	tools, err := client.ListTools()
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("\n✓ Found %d tools:\n", len(tools))
	for i, tool := range tools {
		fmt.Printf("\n[%d] %s\n", i+1, tool.Name)
		if tool.Description != "" {
			fmt.Printf("    Description: %s\n", tool.Description)
		}

		// Show input schema
		if tool.InputSchema != nil {
			fmt.Printf("    Input Schema: %v\n", tool.InputSchema)
		}
	}

	// Try to call the first tool if available
	if len(tools) > 0 {
		fmt.Printf("\n3. Calling first tool: %s\n", tools[0].Name)

		// Call with empty arguments (adjust as needed)
		result, err := client.CallTool(tools[0].Name, map[string]interface{}{})
		if err != nil {
			fmt.Printf("Failed to call tool: %v\n", err)
		} else {
			fmt.Printf("Tool result: %s\n", string(result))
		}
	}

	fmt.Println("\n=== Done ===")
}
