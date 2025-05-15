package client

import (
	"context"
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

	// Configure notification handler
	transportImpl.SetNotificationHandler(func(notification transport.JSONRPCNotification) {
		client := &HTTPClient{
			transport: transportImpl,
			config:    &Config{Options: options},
		}

		if client.notificationHandler != nil {
			client.notificationHandler(notification.Method, notification.Params.AdditionalFields)
		}
	})

	return &HTTPClient{
		transport: transportImpl,
		config:    &Config{Options: options},
	}, nil
}

// Initialize initializes the client with the server.
// This sends the initialize request to the server with the protocol version.
// The server will respond with capabilities and a session ID.
// See: http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#initialization
func (c *HTTPClient) Initialize(ctx context.Context) error {
	// Determine protocol version to use
	protocolVersion := ""
	if c.config != nil && c.config.Options != nil && c.config.Options.ProtocolVersion != "" {
		protocolVersion = c.config.Options.ProtocolVersion
	}

	// Create initialize request with protocol version, clientInfo, and empty capabilities
	request := transport.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": protocolVersion,
			"clientInfo": map[string]interface{}{
				"name":    "mcpgopher",
				"version": Version,
			},
			"capabilities": map[string]interface{}{},
		},
	}

	// Send request
	_, err := c.transport.SendRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	return nil
}

// Close closes the client connection and ends the session with the server.
// See: http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#shutdown
func (c *HTTPClient) Close() error {
	return c.transport.Close()
}

// Request makes a request to the server with custom parameters.
// This is the general-purpose method for sending any MCP method to the server.
// See: http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#requests-and-responses
func (c *HTTPClient) Request(ctx context.Context, method string, params interface{}) ([]byte, error) {
	// Create request
	request := transport.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      time.Now().UnixNano(),
		Method:  method,
		Params:  params,
	}

	// Send request
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

// GetSessionID returns the current session ID assigned by the server.
// The session ID is established during initialization and used for subsequent requests.
// See: http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#sessions
func (c *HTTPClient) GetSessionID() string {
	if httpTransport, ok := c.transport.(*transport.StreamableHTTP); ok {
		return httpTransport.GetSessionId()
	}
	return ""
}

// SetNotificationHandler sets a handler for server notifications.
// Notifications are one-way messages from the server that don't expect a response.
// See: http://spec.modelcontextprotocol.io/2025-03-26/base-protocol#notifications
func (c *HTTPClient) SetNotificationHandler(handler func(method string, params map[string]interface{})) {
	c.notificationHandler = handler
}

// debug logs a message if debug mode is enabled



