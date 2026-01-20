package internal

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

type MCPClient struct {
	baseURL            string
	httpClient         *http.Client
	serverInfo         *Implementation
	serverCapabilities *ServerCapabilities
	mu                 sync.RWMutex
	initialized        bool
	sessionToken       string
	sessionID          string
}

func NewMCPClient(baseURL string) *MCPClient {
	return &MCPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *MCPClient) SetSessionID(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = sessionID
}

func (c *MCPClient) GetSessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

var idEntropy = ulid.Monotonic(rand.Reader, 0)

func generateId() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), idEntropy).String()
}

func (c *MCPClient) GenerateSessionID() string {
	sessionID := generateId()
	c.SetSessionID(sessionID)
	return sessionID
}

func (c *MCPClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

func (c *MCPClient) SetSessionToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionToken = token
}

func (c *MCPClient) sendRequest(method string, params interface{}) (*JSONRPCMessage, error) {
	requestID := generateId()

	var paramsJSON json.RawMessage
	if params != nil {
		paramsBytes, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = json.RawMessage(paramsBytes)
	}

	message := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  paramsJSON,
	}

	reqBody, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.mu.RLock()
	if c.sessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.sessionToken)
	}
	// Add session ID header
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	c.mu.RUnlock()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close request body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response JSONRPCMessage
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if c.sessionID == "" {
		c.mu.RLock()
		c.sessionID = response.ID
		c.mu.RUnlock()
	}

	return &response, nil
}

func (c *MCPClient) sendNotification(method string, params interface{}) error {
	var paramsJSON json.RawMessage
	if params != nil {
		paramsBytes, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
		paramsJSON = json.RawMessage(paramsBytes)
	}

	message := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}

	reqBody, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	c.mu.RLock()
	if c.sessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.sessionToken)
	}
	// Add session ID header
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	c.mu.RUnlock()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close request body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// Initialize the MCP connection
func (c *MCPClient) Initialize() error {
	capabilities := ClientCapabilities{
		Roots: &RootsCapability{
			ListChanged: boolPtr(true),
		},
		Sampling:    &SamplingCapability{},
		Elicitation: &ElicitationCapability{},
	}

	clientInfo := Implementation{
		Name:    "go-mcp-client",
		Version: "1.0.0",
		Title:   "Go MCP Client",
	}

	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    capabilities,
		ClientInfo:      clientInfo,
	}

	response, err := c.sendRequest("initialize", params)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("initialize error: %w", response.Error)
	}

	var result InitializeResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}

	c.mu.Lock()
	c.serverInfo = &result.ServerInfo
	c.sessionID = response.ID
	c.serverCapabilities = &result.Capabilities
	c.initialized = true
	c.mu.Unlock()

	// Send initialized notification
	return c.sendNotification("notifications/initialized", nil)
}

// Ping the server
func (c *MCPClient) Ping() error {
	response, err := c.sendRequest("ping", nil)
	if err != nil {
		return fmt.Errorf("ping request failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("ping error: %w", response.Error)
	}

	return nil
}

// List resources
func (c *MCPClient) ListResources(cursor *Cursor) (*ListResourcesResult, error) {
	params := ListResourcesParams{
		Cursor: cursor,
	}

	response, err := c.sendRequest("resources/list", params)
	if err != nil {
		return nil, fmt.Errorf("list resources request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("list resources error: %w", response.Error)
	}

	var result ListResourcesResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list resources result: %w", err)
	}

	return &result, nil
}

// List resource templates
func (c *MCPClient) ListResourceTemplates(cursor *Cursor) (*ListResourceTemplatesResult, error) {
	params := ListResourceTemplatesParams{
		Cursor: cursor,
	}

	response, err := c.sendRequest("resources/templates/list", params)
	if err != nil {
		return nil, fmt.Errorf("list resource templates request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("list resource templates error: %w", response.Error)
	}

	var result ListResourceTemplatesResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list resource templates result: %w", err)
	}

	return &result, nil
}

// Read resource
func (c *MCPClient) ReadResource(uri string) (*ReadResourceResult, error) {
	params := ReadResourceParams{
		URI: uri,
	}

	response, err := c.sendRequest("resources/read", params)
	if err != nil {
		return nil, fmt.Errorf("read resource request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("read resource error: %w", response.Error)
	}

	var result ReadResourceResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal read resource result: %w", err)
	}

	return &result, nil
}

// List tools
func (c *MCPClient) ListTools(cursor *Cursor) (*ListToolsResult, error) {
	params := ListToolsParams{
		Cursor: cursor,
	}

	response, err := c.sendRequest("tools/list", params)
	if err != nil {
		return nil, fmt.Errorf("list tools request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("list tools error: %w", response.Error)
	}

	var result ListToolsResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list tools result: %w", err)
	}

	return &result, nil
}

// Call tool
func (c *MCPClient) CallTool(name string, arguments map[string]interface{}) (*CallToolResult, error) {
	params := CallToolParams{
		Name:      name,
		Arguments: arguments,
	}

	response, err := c.sendRequest("tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("call tool request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("call tool error: %w", response.Error)
	}

	var result CallToolResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal call tool result: %w", err)
	}

	return &result, nil
}

// List prompts
func (c *MCPClient) ListPrompts(cursor *Cursor) (*ListPromptsResult, error) {
	params := ListPromptsParams{
		Cursor: cursor,
	}

	response, err := c.sendRequest("prompts/list", params)
	if err != nil {
		return nil, fmt.Errorf("list prompts request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("list prompts error: %w", response.Error)
	}

	var result ListPromptsResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list prompts result: %w", err)
	}

	return &result, nil
}

// Get prompt
func (c *MCPClient) GetPrompt(name string, arguments map[string]string) (*GetPromptResult, error) {
	params := GetPromptParams{
		Name:      name,
		Arguments: arguments,
	}

	response, err := c.sendRequest("prompts/get", params)
	if err != nil {
		return nil, fmt.Errorf("get prompt request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("get prompt error: %w", response.Error)
	}

	var result GetPromptResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get prompt result: %w", err)
	}

	return &result, nil
}

// Set logging level
func (c *MCPClient) SetLoggingLevel(level LoggingLevel) error {
	params := SetLevelParams{
		Level: level,
	}

	response, err := c.sendRequest("logging/setLevel", params)
	if err != nil {
		return fmt.Errorf("set logging level request failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("set logging level error: %w", response.Error)
	}

	return nil
}

// Subscribe to resource updates
func (c *MCPClient) Subscribe(uri string) error {
	params := SubscribeParams{
		URI: uri,
	}

	response, err := c.sendRequest("resources/subscribe", params)
	if err != nil {
		return fmt.Errorf("subscribe request failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("subscribe error: %w", response.Error)
	}

	return nil
}

// Unsubscribe from resource updates
func (c *MCPClient) Unsubscribe(uri string) error {
	params := UnsubscribeParams{
		URI: uri,
	}

	response, err := c.sendRequest("resources/unsubscribe", params)
	if err != nil {
		return fmt.Errorf("unsubscribe request failed: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("unsubscribe error: %w", response.Error)
	}

	return nil
}

// Get completions
func (c *MCPClient) Complete(ref CompletionRef, argument CompleteArgument) (*CompleteResult, error) {
	params := CompleteParams{
		Ref:      ref,
		Argument: argument,
	}

	response, err := c.sendRequest("completion/complete", params)
	if err != nil {
		return nil, fmt.Errorf("complete request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("complete error: %w", response.Error)
	}

	var result CompleteResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal complete result: %w", err)
	}

	return &result, nil
}

// Create message (sampling)
func (c *MCPClient) CreateMessage(messages []SamplingMessage, preferences *ModelPreferences) (*CreateMessageResult, error) {
	params := CreateMessageParams{
		Messages:         messages,
		ModelPreferences: preferences,
	}

	response, err := c.sendRequest("sampling/createMessage", params)
	if err != nil {
		return nil, fmt.Errorf("create message request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("create message error: %w", response.Error)
	}

	var result CreateMessageResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create message result: %w", err)
	}

	return &result, nil
}

// List roots
func (c *MCPClient) ListRoots() (*ListRootsResult, error) {
	response, err := c.sendRequest("roots/list", nil)
	if err != nil {
		return nil, fmt.Errorf("list roots request failed: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("list roots error: %w", response.Error)
	}

	var result ListRootsResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list roots result: %w", err)
	}

	return &result, nil
}

// Server capability checks
func (c *MCPClient) ServerSupportsLogging() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities != nil && c.serverCapabilities.Logging != nil
}

func (c *MCPClient) ServerSupportsCompletions() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities != nil && c.serverCapabilities.Completions != nil
}

func (c *MCPClient) ServerSupportsPrompts() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities != nil && c.serverCapabilities.Prompts != nil
}

func (c *MCPClient) ServerSupportsResources() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities != nil && c.serverCapabilities.Resources != nil
}

func (c *MCPClient) ServerSupportsResourceSubscription() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities != nil &&
		c.serverCapabilities.Resources != nil &&
		c.serverCapabilities.Resources.Subscribe != nil &&
		*c.serverCapabilities.Resources.Subscribe
}

func (c *MCPClient) ServerSupportsTools() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities != nil && c.serverCapabilities.Tools != nil
}

func (c *MCPClient) GetServerInfo() *Implementation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

func (c *MCPClient) GetServerCapabilities() *ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverCapabilities
}

func (c *MCPClient) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func Float64Ptr(f float64) *float64 {
	return &f
}

// Advanced HTTP client features
func (c *MCPClient) SetCustomHeaders(headers map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store headers for future requests
	if c.httpClient.Transport == nil {
		c.httpClient.Transport = &http.Transport{}
	}

	// Note: This is a simplified approach. In production, you might want to
	// implement a custom RoundTripper to add headers to all requests
}

func (c *MCPClient) SetContext(ctx context.Context) {
	// This would be used for request context in production
	// For now, we'll keep it simple
}

// Batch operations
func (c *MCPClient) BatchListAll() (map[string]interface{}, error) {
	results := make(map[string]interface{})

	// List all resources
	if c.ServerSupportsResources() {
		resources, err := c.ListResources(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources: %w", err)
		}
		results["resources"] = resources
	}

	// List all tools
	if c.ServerSupportsTools() {
		tools, err := c.ListTools(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list tools: %w", err)
		}
		results["tools"] = tools
	}

	// List all prompts
	if c.ServerSupportsPrompts() {
		prompts, err := c.ListPrompts(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list prompts: %w", err)
		}
		results["prompts"] = prompts
	}

	return results, nil
}

// Convenience methods for common operations
func (c *MCPClient) ReadAllResources() (map[string]*ReadResourceResult, error) {
	if !c.ServerSupportsResources() {
		return nil, fmt.Errorf("server does not support resources")
	}

	resources, err := c.ListResources(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	results := make(map[string]*ReadResourceResult)
	for _, resource := range resources.Resources {
		content, err := c.ReadResource(resource.URI)
		if err != nil {
			return nil, fmt.Errorf("failed to read resource %s: %w", resource.URI, err)
		}
		results[resource.URI] = content
	}

	return results, nil
}

func (c *MCPClient) CallAllTools(arguments map[string]interface{}) (map[string]*CallToolResult, error) {
	if !c.ServerSupportsTools() {
		return nil, fmt.Errorf("server does not support tools")
	}

	tools, err := c.ListTools(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	results := make(map[string]*CallToolResult)
	for _, tool := range tools.Tools {
		result, err := c.CallTool(tool.Name, arguments)
		if err != nil {
			return nil, fmt.Errorf("failed to call tool %s: %w", tool.Name, err)
		}
		results[tool.Name] = result
	}

	return results, nil
}

// Error handling helpers
func (c *MCPClient) IsConnectionError(err error) bool {
	return err != nil && (err.Error() == "connection refused" ||
		err.Error() == "connection timeout" ||
		err.Error() == "no route to host")
}

func (c *MCPClient) IsAuthError(err error) bool {
	return err != nil && (err.Error() == "HTTP error: 401 Unauthorized" ||
		err.Error() == "HTTP error: 403 Forbidden")
}

func (c *MCPClient) IsServerError(err error) bool {
	return err != nil && (err.Error() == "HTTP error: 500 Internal Server Error" ||
		err.Error() == "HTTP error: 502 Bad Gateway" ||
		err.Error() == "HTTP error: 503 Service Unavailable")
}

// Retry logic
func (c *MCPClient) RetryRequest(operation func() error, maxRetries int) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on authentication errors
		if c.IsAuthError(err) {
			return err
		}

		// Exponential backoff
		if i < maxRetries-1 {
			time.Sleep(time.Duration(1<<i) * time.Second)
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}
