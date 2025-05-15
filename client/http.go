package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/contriboss/mcpgopher/client/transport"
)

// HTTPClient implements the Interface for MCP client over HTTP transport.
// It implements the Model Context Protocol (MCP) client-side functionality.
// See: http://spec.modelcontextprotocol.io/2025-03-26/
type HTTPClient struct {
	transport transport.Interface
	config    *Config

	notificationHandler func(method string, params map[string]interface{})
}

// NewHTTPClient creates a new HTTP client
func NewHTTPClient(options *Options) (*HTTPClient, error) {
	if options == nil {
		options = &Options{}
	}

	// Set default base URL if not provided
	if options.BaseURL == "" {
		options.BaseURL = "http://localhost:62770"
	}

	// Create transport options
	transportOpts := []transport.StreamableHTTPCOption{}

	// Add headers if provided
	if len(options.Headers) > 0 {
		transportOpts = append(transportOpts, transport.WithHTTPHeaders(options.Headers))
	}

	// Add timeout if provided
	if options.Timeout > 0 {
		transportOpts = append(transportOpts, transport.WithHTTPTimeout(time.Duration(options.Timeout)*time.Second))
	}

	// Create transport
	transportImpl, err := transport.NewStreamableHTTP(options.BaseURL, transportOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	client := &HTTPClient{
		transport: transportImpl,
		config:    &Config{Options: options},
	}

	// Configure notification handler
	transportImpl.SetNotificationHandler(func(notification transport.JSONRPCNotification) {
		if client.notificationHandler != nil {
			client.notificationHandler(notification.Method, notification.Params.AdditionalFields)
		}
	})

	// Immediately initialize the transport (connect to server)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Determine protocol version, clientInfo, and capabilities
	protocolVersion := "2025-03-26"
	if client.config != nil && client.config.Options != nil && client.config.Options.ProtocolVersion != "" {
		protocolVersion = client.config.Options.ProtocolVersion
	}
	clientInfo := map[string]interface{}{
		"name":    "mcpgopher",
		"version": Version,
	}
	capabilities := map[string]interface{}{}

	if err := transportImpl.Initialize(ctx, protocolVersion, clientInfo, capabilities); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	return client, nil
}

// Initialize initializes the client with the server using the transport's Initialize method.
func (c *HTTPClient) Initialize(ctx context.Context) error {
	protocolVersion := "2025-03-26"
	if c.config != nil && c.config.Options != nil && c.config.Options.ProtocolVersion != "" {
		protocolVersion = c.config.Options.ProtocolVersion
	}
	clientInfo := map[string]interface{}{
		"name":    "mcpgopher",
		"version": Version,
	}
	capabilities := map[string]interface{}{}
	return c.transport.Initialize(ctx, protocolVersion, clientInfo, capabilities)
}

// Close closes the client connection and ends the session with the server.
// See: http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#shutdown
func (c *HTTPClient) Close() error {
	return c.transport.Close()
}

// SetNotificationHandler sets a handler for server notifications.
// Notifications are one-way messages from the server that don't expect a response.
// See: http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#notifications
func (c *HTTPClient) SetNotificationHandler(handler func(method string, params map[string]interface{})) {
	c.notificationHandler = handler
}

// Request makes a request to the server with custom parameters.
// This is the general-purpose method for sending any MCP method to the server.
// See: http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#requests-and-responses
func (c *HTTPClient) Request(ctx context.Context, method string, params interface{}) ([]byte, error) {
	// Ensure initialize always sends required params
	if method == "initialize" {
		if params == nil {
			protocolVersion := "2025-03-26"
			if c.config != nil && c.config.Options != nil && c.config.Options.ProtocolVersion != "" {
				protocolVersion = c.config.Options.ProtocolVersion
			}
			clientInfo := map[string]interface{}{
				"name":    "mcpgopher",
				"version": Version,
			}
			params = map[string]interface{}{
				"protocolVersion": protocolVersion,
				"clientInfo":      clientInfo,
				"capabilities":    map[string]interface{}{},
			}
		}
	}

	// Create the JSONRPC request
	request := transport.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Method:  method,
		Params:  params,
	}

	// Send request using the transport interface
	response, err := c.transport.SendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for error
	if response.Error != nil {
		return nil, fmt.Errorf("error %d: %s", response.Error.Code, response.Error.Message)
	}

	return response.Result, nil
}

// GetSessionID returns the current session ID
func (c *HTTPClient) GetSessionID() string {
	if t, ok := c.transport.(*transport.StreamableHTTP); ok {
		return t.GetSessionId()
	}
	return ""
}

// Ping sends a ping request to the server and waits for a response.
// It returns an error if the ping fails.
// See: http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#ping
func (c *HTTPClient) Ping(ctx context.Context) error {
	return c.transport.Ping(ctx)
}

// RawRequest sends a request and returns the full JSON-RPC envelope as bytes.
func (c *HTTPClient) RawRequest(ctx context.Context, method string, params interface{}) ([]byte, error) {
	request := transport.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Method:  method,
		Params:  params,
	}
	response, err := c.transport.SendRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	return json.Marshal(response)
}
