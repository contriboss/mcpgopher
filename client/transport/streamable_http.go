package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oklog/ulid"
)

type StreamableHTTPCOption func(*StreamableHTTP)

func WithHTTPHeaders(headers map[string]string) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.headers = headers
	}
}

// WithHTTPTimeout sets the timeout for a HTTP request and stream.
func WithHTTPTimeout(timeout time.Duration) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.httpClient.Timeout = timeout
	}
}

// StreamableHTTP implements Streamable HTTP transport.
//
// It transmits JSON-RPC messages over individual HTTP requests. One message per request.
// The HTTP response body can either be a single JSON-RPC response,
// or an upgraded SSE stream that concludes with a JSON-RPC response for the same request.
//
// http://spec.modelcontextprotocol.io/2025-03-26/base-protocol
//
// The current implementation does not support the following features:
//   - batching
//   - continuously listening for server notifications when no request is in flight
//     (http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#transport)
//   - resuming stream
//     (http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#transport)
//   - server -> client request
type StreamableHTTP struct {
	baseURL    *url.URL
	httpClient *http.Client
	headers    map[string]string

	sessionID   atomic.Value
	initialized atomic.Bool

	notificationHandler func(JSONRPCNotification)
	notifyMu            sync.RWMutex

	closed chan struct{}
}

// NewStreamableHTTP creates a new Streamable HTTP transport with the given base URL.
// Returns an error if the URL is invalid.
func NewStreamableHTTP(baseURL string, options ...StreamableHTTPCOption) (*StreamableHTTP, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	smc := &StreamableHTTP{
		baseURL:    parsedURL,
		httpClient: &http.Client{},
		headers:    make(map[string]string),
		closed:     make(chan struct{}),
	}
	smc.sessionID.Store("") // set initial value to simplify later usage

	for _, opt := range options {
		opt(smc)
	}

	return smc, nil
}

// Start initiates the HTTP connection to the server.
func (c *StreamableHTTP) Start(ctx context.Context) error {
	// For Streamable HTTP, we don't need to establish a persistent connection
	return nil
}

// Initialize sends the initialize request to the server with protocol version, client info, and capabilities.
// Stores the session ID if successful.
func (c *StreamableHTTP) Initialize(ctx context.Context, protocolVersion string, clientInfo map[string]interface{}, capabilities map[string]interface{}) error {
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  initializeMethod,
		Params: map[string]interface{}{
			"protocolVersion": protocolVersion,
			"clientInfo":      clientInfo,
			"capabilities":    capabilities,
		},
	}

	_, err := c.SendRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Note: The sessionID is already stored in SendRequest when processing
	// the HTTP headers for the initialize method

	c.initialized.Store(true)
	return nil
}

// Close closes the all the HTTP connections to the server.
func (c *StreamableHTTP) Close() error {
	select {
	case <-c.closed:
		return nil
	default:
	}
	// Cancel all in-flight requests
	close(c.closed)

	sessionId := c.sessionID.Load().(string)
	if sessionId != "" {
		c.sessionID.Store("")

		// notify server session closed
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL.String(), nil)
			if err != nil {
				fmt.Printf("failed to create close request\n: %v", err)
				return
			}
			req.Header.Set(headerKeySessionID, sessionId)
			res, err := c.httpClient.Do(req)
			if err != nil {
				fmt.Printf("failed to send close request\n: %v", err)
				return
			}
			res.Body.Close()
		}()
	}

	return nil
}

const (
	initializeMethod   = "initialize"
	headerKeySessionID = "Mcp-Session-Id"
)

// SendRequest sends a JSON-RPC request to the server and waits for a response.
// Returns the raw JSON response message or an error if the request fails.
func (c *StreamableHTTP) SendRequest(
	ctx context.Context,
	request JSONRPCRequest,
) (*JSONRPCResponse, error) {
	// Print debug info for ping requests
	if request.Method == "ping" {
		fmt.Printf("DEBUG SendRequest: Method=%s, ID=%s\n", request.Method, request.ID)
	}

	// Create a combined context that could be canceled when the client is closed
	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-c.closed:
			cancel()
		case <-newCtx.Done():
			// The original context was canceled, no need to do anything
		}
	}()
	ctx = newCtx

	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String(), bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	sessionID := c.sessionID.Load()
	if sessionID != "" {
		req.Header.Set(headerKeySessionID, sessionID.(string))
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check if we got an error response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		// handle session closed
		if resp.StatusCode == http.StatusNotFound {
			c.sessionID.CompareAndSwap(sessionID, "")
			return nil, fmt.Errorf("session terminated (404). need to re-initialize")
		}

		// handle error response
		var errResponse JSONRPCResponse
		body, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(body, &errResponse); err == nil {
			return &errResponse, nil
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, body)
	}

	if request.Method == initializeMethod {
		// saved the received session ID in the response
		// empty session ID is allowed
		if sessionID := resp.Header.Get(headerKeySessionID); sessionID != "" {
			c.sessionID.Store(sessionID)
		}
	}

	// Handle different response types
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	switch mediaType {
	case "application/json":
		// Single response
		body, _ := io.ReadAll(resp.Body)
		
		// Log the raw response for debugging if it's a ping
		if request.Method == "ping" {
			fmt.Printf("DEBUG Raw response: %s\n", string(body))
		}
		
		var response JSONRPCResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w\nRaw payload: %s", err, string(body))
		}

		// Special handling for ping requests - allow null ID
		if response.ID == nil && request.Method != "ping" {
			return nil, fmt.Errorf("response should contain RPC id. Raw payload: %s", string(body))
		}

		return &response, nil

	case "text/event-stream":
		// Server is using SSE for streaming responses
		return c.handleSSEResponse(ctx, resp.Body)

	default:
		return nil, fmt.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
	}
}

func (c *StreamableHTTP) Request(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String(),
		Method:  method,
		Params:  params,
	}

	return c.SendRequest(ctx, request)
}

// handleSSEResponse processes an SSE stream for a specific request.
// It returns the final result for the request once received, or an error.
func (c *StreamableHTTP) handleSSEResponse(ctx context.Context, reader io.ReadCloser) (*JSONRPCResponse, error) {

	// Create a channel for this specific request
	responseChan := make(chan *JSONRPCResponse, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start a goroutine to process the SSE stream
	go func() {
		// only close responseChan after readingSSE()
		defer close(responseChan)

		c.readSSE(ctx, reader, func(event, data string) {

			// (unsupported: batching)

			var message JSONRPCResponse
			if err := json.Unmarshal([]byte(data), &message); err != nil {
				fmt.Printf("failed to unmarshal message: %v\n", err)
				return
			}

			// Handle notification
			if message.ID == nil {
				var notification JSONRPCNotification
				if err := json.Unmarshal([]byte(data), &notification); err != nil {
					fmt.Printf("failed to unmarshal notification: %v\n", err)
					return
				}
				c.notifyMu.RLock()
				if c.notificationHandler != nil {
					c.notificationHandler(notification)
				}
				c.notifyMu.RUnlock()
				return
			}

			responseChan <- &message
		})
	}()

	// Wait for the response or context cancellation
	select {
	case response := <-responseChan:
		if response == nil {
			return nil, fmt.Errorf("unexpected nil response")
		}
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// readSSE reads the SSE stream(reader) and calls the handler for each event and data pair.
// It will end when the reader is closed (or the context is done).
func (c *StreamableHTTP) readSSE(ctx context.Context, reader io.ReadCloser, handler func(event, data string)) {
	defer reader.Close()

	br := bufio.NewReader(reader)
	var event, data string

	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := br.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Process any pending event before exit
					if event != "" && data != "" {
						handler(event, data)
					}
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
					fmt.Printf("SSE stream error: %v\n", err)
					return
				}
			}

			// Remove only newline markers
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				// Empty line means end of event
				if event != "" && data != "" {
					handler(event, data)
					event = ""
					data = ""
				}
				continue
			}

			if strings.HasPrefix(line, "event:") {
				event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}
	}
}

func (c *StreamableHTTP) SendNotification(ctx context.Context, notification JSONRPCNotification) error {

	// Marshal request
	requestBody, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String(), bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID := c.sessionID.Load(); sessionID != "" {
		req.Header.Set(headerKeySessionID, sessionID.(string))
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"notification failed with status %d: %s",
			resp.StatusCode,
			body,
		)
	}

	return nil
}

func (c *StreamableHTTP) SetNotificationHandler(handler func(JSONRPCNotification)) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	c.notificationHandler = handler
}

func (c *StreamableHTTP) GetSessionId() string {
	return c.sessionID.Load().(string)
}

// Ping sends a ping request to the server and waits for a response.
// This can be used to check if the server is still alive and measure latency.
func (c *StreamableHTTP) Ping(ctx context.Context) error {
	// For ping request
	pingParams := map[string]interface{}{
		"timestamp": time.Now().UnixNano(),
	}
	
	// Create request ID for ping
	requestID := fmt.Sprintf("ping-%d", time.Now().UnixNano())
	fmt.Printf("DEBUG: Using request ID: %s\n", requestID)
	
	// Try using SendRequest instead of direct HTTP request
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  "ping",
		Params:  pingParams,
	}
	
	// Marshal request for logging
	requestBody, _ := json.Marshal(request)
	fmt.Printf("DEBUG: Sending ping request: %s\n", string(requestBody))
	
	// Send the ping request
	resp, err := c.SendRequest(ctx, request)
	if err != nil {
		fmt.Printf("DEBUG: Ping error: %v\n", err)
		return fmt.Errorf("ping failed: %w", err)
	}
	
	// Log response
	respJSON, _ := json.Marshal(resp)
	fmt.Printf("DEBUG: Ping response: %s\n", string(respJSON))
	
	return nil
}
