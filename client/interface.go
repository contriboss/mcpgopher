package client

import (
	"context"
	"io"
)

// Interface for MCP client
type Interface interface {
	// Initialize initializes the client with the server
	Initialize(ctx context.Context) error
	
	// Close closes the client connection
	Close() error
	
	// Request makes a request to the server with custom parameters
	Request(ctx context.Context, method string, params interface{}) ([]byte, error)
	
	// SendInput sends input to the server
	SendInput(ctx context.Context, input string) error
	
	// GetSessionID returns the current session ID
	GetSessionID() string
	
	// SetNotificationHandler sets a handler for server notifications
	SetNotificationHandler(handler func(method string, params map[string]interface{}))
}

// Options for client configuration
type Options struct {
	// BaseURL is the server URL to connect to
	BaseURL string
	
	// Headers are additional HTTP headers to include in requests
	Headers map[string]string
	
	// Timeout is the request timeout
	Timeout int
	
	// Debug enables debug logging
	Debug bool
	
	// Logger provides a custom logger
	Logger io.Writer
	
	// ProtocolVersion specifies the MCP protocol version to use
	// If not provided, defaults to "2025-03-26"
	ProtocolVersion string
	
	// Capabilities defines the client capabilities to advertise to the server
	// If not provided, default capabilities will be used
	Capabilities map[string]interface{}
}

// Config represents client configuration
type Config struct {
	// Options contains user-provided configuration
	Options *Options
}