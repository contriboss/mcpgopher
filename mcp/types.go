// Package mcp defines the core types and interfaces for the Model Context Protocol (MCP).
// MCP enables seamless integration between LLM applications and their supporting services.
package mcp

import (
	"encoding/json"

	"github.com/yosida95/uritemplate/v3"
)

// MCPMethod represents a protocol method identifier
type MCPMethod string

const (
	// MethodInitialize negotiates protocol capabilities and version
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/lifecycle/#initialization
	MethodInitialize MCPMethod = "initialize"

	// MethodPing validates connection liveness
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/utilities/ping
	MethodPing MCPMethod = "ping"

	// MethodResourcesList retrieves available server resources
	// https://modelcontextprotocol.io/specification/2025-03-26/server/resources
	MethodResourcesList MCPMethod = "resources/list"

	// MethodResourcesTemplatesList retrieves URI templates for resource construction
	// https://modelcontextprotocol.io/specification/2025-03-26/server/resources
	MethodResourcesTemplatesList MCPMethod = "resources/templates/list"

	// MethodResourcesRead fetches specific resource content
	// https://modelcontextprotocol.io/specification/2025-03-26/server/resources
	MethodResourcesRead MCPMethod = "resources/read"

	// MethodPromptsList retrieves available prompt templates
	// https://modelcontextprotocol.io/specification/2025-03-26/server/prompts
	MethodPromptsList MCPMethod = "prompts/list"

	// MethodPromptsGet fetches prompt with filled arguments
	// https://modelcontextprotocol.io/specification/2025-03-26/server/prompts
	MethodPromptsGet MCPMethod = "prompts/get"

	// MethodToolsList retrieves available executable tools
	// https://modelcontextprotocol.io/specification/2025-03-26/server/tools
	MethodToolsList MCPMethod = "tools/list"

	// MethodToolsCall executes a tool with provided arguments
	// https://modelcontextprotocol.io/specification/2025-03-26/server/tools
	MethodToolsCall MCPMethod = "tools/call"

	// MethodCompleteList provides argument completion suggestions
	// https://modelcontextprotocol.io/specification/2025-03-26/utilities/completion
	MethodCompleteList MCPMethod = "completion/complete"

	// MethodLoggingSetLevel adjusts server logging verbosity
	// https://modelcontextprotocol.io/specification/2025-03-26/utilities/logging
	MethodLoggingSetLevel MCPMethod = "logging/setLevel"

	// MethodRootsList retrieves available file system roots
	// https://modelcontextprotocol.io/specification/2025-03-26/client/roots
	MethodRootsList MCPMethod = "roots/list"

	// MethodSamplingCreateMessage requests LLM sampling via client
	// https://modelcontextprotocol.io/specification/2025-03-26/client/sampling
	MethodSamplingCreateMessage MCPMethod = "sampling/createMessage"

	// MethodNotificationInitialized confirms initialization complete
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/lifecycle
	MethodNotificationInitialized MCPMethod = "notifications/initialized"

	// MethodNotificationCancelled indicates request cancellation
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/cancellation
	MethodNotificationCancelled MCPMethod = "notifications/cancelled"

	// MethodNotificationProgress provides operation progress updates
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/progress
	MethodNotificationProgress MCPMethod = "notifications/progress"

	// MethodNotificationResourcesListChanged signals resource list changes
	// https://modelcontextprotocol.io/specification/2025-03-26/server/resources#list-changed-notification
	MethodNotificationResourcesListChanged MCPMethod = "notifications/resources/list_changed"

	// MethodNotificationResourceUpdated signals specific resource changes
	// https://modelcontextprotocol.io/specification/2025-03-26/server/resources#updated-notification
	MethodNotificationResourceUpdated MCPMethod = "notifications/resources/updated"

	// MethodNotificationPromptsListChanged signals prompt list changes
	// https://modelcontextprotocol.io/specification/2025-03-26/server/prompts#list-changed-notification
	MethodNotificationPromptsListChanged MCPMethod = "notifications/prompts/list_changed"

	// MethodNotificationToolsListChanged signals tool list changes
	// https://modelcontextprotocol.io/specification/2025-03-26/server/tools#list-changed-notification
	MethodNotificationToolsListChanged MCPMethod = "notifications/tools/list_changed"

	// MethodNotificationLoggingMessage transmits log entries
	// https://modelcontextprotocol.io/specification/2025-03-26/utilities/logging
	MethodNotificationLoggingMessage MCPMethod = "notifications/logging/message"

	// MethodNotificationRootsListChanged signals root list changes
	// https://modelcontextprotocol.io/specification/2025-03-26/client/roots
	MethodNotificationRootsListChanged MCPMethod = "notifications/roots/list_changed"
)

// URITemplate wraps URI template functionality for JSON serialization
type URITemplate struct {
	*uritemplate.Template
}

func (t *URITemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Raw())
}

func (t *URITemplate) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	template, err := uritemplate.New(raw)
	if err != nil {
		return err
	}
	t.Template = template
	return nil
}

/* Constants */

// LATEST_PROTOCOL_VERSION defines the current MCP protocol version
const LATEST_PROTOCOL_VERSION = "2025-03-26"

// JSONRPC_VERSION specifies the JSON-RPC version used by MCP
const JSONRPC_VERSION = "2.0"

/* Core Types */

// ProgressToken associates progress notifications with original requests
type ProgressToken interface{}

// Cursor represents an opaque pagination token
type Cursor string

// RequestId uniquely identifies a JSON-RPC request
type RequestId interface{}

// JSONRPCMessage encompasses all JSON-RPC message types
type JSONRPCMessage interface{}

// JSONRPCRequest represents a request expecting a response
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      RequestId              `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// JSONRPCNotification represents a message not expecting a response
type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a successful response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      RequestId   `json:"id"`
	Result  interface{} `json:"result,omitempty"`
}

// JSONRPCError represents an error response
type JSONRPCError struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      RequestId `json:"id"`
	Error   struct {
		// Error code identifying the error type
		Code int `json:"code"`
		// Concise error description
		Message string `json:"message"`
		// Additional error details
		Data interface{} `json:"data,omitempty"`
	} `json:"error"`
}

// Standard error codes
const (
	// JSON-RPC standard errors
	ErrorParseError     = -32700
	ErrorInvalidRequest = -32600
	ErrorMethodNotFound = -32601
	ErrorInvalidParams  = -32602
	ErrorInternalError  = -32603

	// MCP-specific errors
	ErrorResourceNotFound = -32002
	ErrorToolNotFound     = -32003
	ErrorUnauthorized     = -32004
	ErrorDuplicateName    = -32005
	ErrorInvalidProtocol  = -32006
)

// Result represents a successful operation result
type Result struct {
	// Protocol-reserved metadata field
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// EmptyResult indicates successful completion without data
type EmptyResult struct {
	Result
}

// Notification represents a notification message
type Notification struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params,omitempty"`
}

/* Initialization */

// InitializeRequest initiates protocol handshake
type InitializeRequest struct {
	Method string `json:"method"`
	Params struct {
		// Maximum protocol version supported by client
		ProtocolVersion string             `json:"protocolVersion"`
		Capabilities    ClientCapabilities `json:"capabilities"`
		ClientInfo      Implementation     `json:"clientInfo"`
	} `json:"params"`
}

// InitializeResult completes protocol handshake
type InitializeResult struct {
	// Protocol version server will use
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	// Usage instructions for LLM understanding
	Instructions string `json:"instructions,omitempty"`
}

// InitializedNotification confirms initialization complete
type InitializedNotification struct {
	Notification
}

// Implementation identifies an MCP implementation
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

/* Capabilities */

// ClientCapabilities declares client features
type ClientCapabilities struct {
	// Experimental capabilities
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Root listing support
	Roots *RootsCapabilities `json:"roots,omitempty"`
	// Server-initiated sampling support
	Sampling *SamplingCapabilities `json:"sampling,omitempty"`
}

// ServerCapabilities declares server features
type ServerCapabilities struct {
	// Experimental capabilities
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Logging support
	Logging *LoggingCapabilities `json:"logging,omitempty"`
	// Prompt template support
	Prompts *PromptsCapabilities `json:"prompts,omitempty"`
	// Resource support
	Resources *ResourcesCapabilities `json:"resources,omitempty"`
	// Tool support
	Tools *ToolsCapabilities `json:"tools,omitempty"`
}

// RootsCapabilities defines root listing capabilities
type RootsCapabilities struct {
	// Supports change notifications
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapabilities defines sampling capabilities
type SamplingCapabilities struct {
	// Supported sampling features
	Features map[string]interface{} `json:"features,omitempty"`
}

// LoggingCapabilities defines logging capabilities
type LoggingCapabilities struct {
	// Supported log levels
	Levels []LoggingLevel `json:"levels,omitempty"`
}

// PromptsCapabilities defines prompt capabilities
type PromptsCapabilities struct {
	// Supports change notifications
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapabilities defines resource capabilities
type ResourcesCapabilities struct {
	// Supports resource subscriptions
	Subscribe bool `json:"subscribe,omitempty"`
	// Supports change notifications
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolsCapabilities defines tool capabilities
type ToolsCapabilities struct {
	// Supports change notifications
	ListChanged bool `json:"listChanged,omitempty"`
}

/* Annotations */

// Annotations provides metadata for clients
type Annotations struct {
	// Target audiences for this content
	Audience []Role `json:"audience,omitempty"`
	// Importance level (1 = most important, 0 = least)
	Priority float64 `json:"priority,omitempty"`
}

// Annotated embeds optional annotations
type Annotated struct {
	Annotations *Annotations `json:"annotations,omitempty"`
}

/* Resources */

// Resource represents available server data
type Resource struct {
	Annotated
	// Resource identifier
	URI string `json:"uri"`
	// Human-readable name
	Name string `json:"name"`
	// Purpose description
	Description string `json:"description,omitempty"`
	// Content MIME type
	MimeType string `json:"mimeType,omitempty"`
	// Raw content size in bytes
	Size *int64 `json:"size,omitempty"`
}

// ResourceTemplate defines parameterized resource URIs
type ResourceTemplate struct {
	Annotated
	// URI template following RFC 6570
	URITemplate *URITemplate `json:"uriTemplate"`
	// Human-readable name
	Name string `json:"name"`
	// Purpose description
	Description string `json:"description,omitempty"`
	// Content MIME type (if uniform)
	MimeType string `json:"mimeType,omitempty"`
}

// ResourceContents represents resource content
type ResourceContents interface {
	isResourceContents()
}

// TextResourceContents contains text content
type TextResourceContents struct {
	// Resource identifier
	URI string `json:"uri"`
	// Content MIME type
	MimeType string `json:"mimeType,omitempty"`
	// Text content
	Text string `json:"text"`
}

func (TextResourceContents) isResourceContents() {}

// BlobResourceContents contains binary content
type BlobResourceContents struct {
	// Resource identifier
	URI string `json:"uri"`
	// Content MIME type
	MimeType string `json:"mimeType,omitempty"`
	// Base64-encoded binary data
	Blob string `json:"blob"`
}

func (BlobResourceContents) isResourceContents() {}

// EmbeddedResource embeds resource content inline
type EmbeddedResource struct {
	Type     string           `json:"type"`
	Resource ResourceContents `json:"resource"`
}

func (EmbeddedResource) isContent() {}

/* Tools */

// Tool represents an executable function
type Tool struct {
	Annotated
	// Unique tool identifier
	Name string `json:"name"`
	// Human-readable description
	Description string `json:"description,omitempty"`
	// JSON Schema for parameters
	InputSchema json.RawMessage `json:"inputSchema"`
	// Behavior hints for clients
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

// ToolAnnotations provides behavioral hints
type ToolAnnotations struct {
	// Display title
	Title string `json:"title,omitempty"`
	// Indicates read-only operation
	ReadOnlyHint *bool `json:"readOnlyHint,omitempty"`
	// Indicates destructive operation
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	// Indicates idempotent operation
	IdempotentHint *bool `json:"idempotentHint,omitempty"`
	// Indicates open-world interaction
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
}

/* Prompts */

// Prompt represents a template for LLM interactions
type Prompt struct {
	Annotated
	// Unique prompt identifier
	Name string `json:"name"`
	// Human-readable description
	Description string `json:"description,omitempty"`
	// Template arguments
	Arguments []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument defines a prompt parameter
type PromptArgument struct {
	// Argument name
	Name string `json:"name"`
	// Argument description
	Description string `json:"description,omitempty"`
	// Whether argument is mandatory
	Required bool `json:"required"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    Role    `json:"role"`
	Content Content `json:"content"`
}

/* Content Types */

// Content represents message content
type Content interface {
	isContent()
}

// TextContent represents text data
type TextContent struct {
	Annotated
	Type string `json:"type"` // Must be "text"
	Text string `json:"text"`
}

func (TextContent) isContent() {}

// ImageContent represents image data
type ImageContent struct {
	Annotated
	Type     string `json:"type"` // Must be "image"
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

func (ImageContent) isContent() {}

// AudioContent represents audio data
type AudioContent struct {
	Annotated
	Type     string `json:"type"` // Must be "audio"
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

func (AudioContent) isContent() {}

/* Roles */

// Role identifies message participants
type Role string

const (
	RoleAssistant Role = "assistant"
	RoleUser      Role = "user"
	RoleSystem    Role = "system"
)

/* Sampling */

// CreateMessageRequest initiates AI sampling
type CreateMessageRequest struct {
	Method string `json:"method"`
	Params struct {
		Messages         []SamplingMessage `json:"messages"`
		ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
		SystemPrompt     string            `json:"systemPrompt,omitempty"`
		IncludeContext   string            `json:"includeContext,omitempty"`
		Temperature      float64           `json:"temperature,omitempty"`
		MaxTokens        int               `json:"maxTokens"`
		StopSequences    []string          `json:"stopSequences,omitempty"`
		Metadata         interface{}       `json:"metadata,omitempty"`
	} `json:"params"`
}

// CreateMessageResult contains sampling response
type CreateMessageResult struct {
	Result
	SamplingMessage
	// Model used for generation
	Model string `json:"model"`
	// Reason sampling stopped
	StopReason string `json:"stopReason,omitempty"`
}

// SamplingMessage represents an LLM message
type SamplingMessage struct {
	Role    Role    `json:"role"`
	Content Content `json:"content"`
}

// ModelPreferences guides model selection
type ModelPreferences struct {
	// Ordered selection hints
	Hints []ModelHint `json:"hints,omitempty"`
	// Cost importance (0-1)
	CostPriority float64 `json:"costPriority,omitempty"`
	// Speed importance (0-1)
	SpeedPriority float64 `json:"speedPriority,omitempty"`
	// Capability importance (0-1)
	IntelligencePriority float64 `json:"intelligencePriority,omitempty"`
}

// ModelHint suggests model characteristics
type ModelHint struct {
	// Model name substring
	Name string `json:"name,omitempty"`
}

/* Pagination */

// PaginatedRequest supports result pagination
type PaginatedRequest struct {
	// Pagination cursor
	Cursor Cursor `json:"cursor,omitempty"`
}

// PaginatedResult provides pagination support
type PaginatedResult struct {
	// Next page cursor
	NextCursor Cursor `json:"nextCursor,omitempty"`
}

/* Logging */

// LoggingLevel represents message severity
type LoggingLevel string

const (
	LoggingLevelDebug     LoggingLevel = "debug"
	LoggingLevelInfo      LoggingLevel = "info"
	LoggingLevelNotice    LoggingLevel = "notice"
	LoggingLevelWarning   LoggingLevel = "warning"
	LoggingLevelError     LoggingLevel = "error"
	LoggingLevelCritical  LoggingLevel = "critical"
	LoggingLevelAlert     LoggingLevel = "alert"
	LoggingLevelEmergency LoggingLevel = "emergency"
)

// SetLevelRequest adjusts logging verbosity
type SetLevelRequest struct {
	Method string `json:"method"`
	Params struct {
		// Minimum severity to log
		Level LoggingLevel `json:"level"`
	} `json:"params"`
}

// LoggingMessageNotification transmits log entries
type LoggingMessageNotification struct {
	Notification
	Params struct {
		// Message severity
		Level LoggingLevel `json:"level"`
		// Logger name
		Logger string `json:"logger,omitempty"`
		// Log content
		Data interface{} `json:"data"`
	} `json:"params"`
}

/* Progress */

// ProgressNotification reports operation progress
type ProgressNotification struct {
	Notification
	Params struct {
		// Associated request token
		ProgressToken ProgressToken `json:"progressToken"`
		// Current progress value
		Progress float64 `json:"progress"`
		// Total expected value
		Total float64 `json:"total,omitempty"`
		// Status description
		Message string `json:"message,omitempty"`
	} `json:"params"`
}

/* Cancellation */

// CancelledNotification signals request cancellation
type CancelledNotification struct {
	Notification
	Params struct {
		// Request ID to cancel
		RequestId RequestId `json:"requestId"`
		// Cancellation reason
		Reason string `json:"reason,omitempty"`
	} `json:"params"`
}

/* Roots */

// Root represents a file system access point
type Root struct {
	// Root URI (must start with file://)
	URI string `json:"uri"`
	// Human-readable name
	Name string `json:"name"`
}

// ListRootsRequest queries available roots
type ListRootsRequest struct {
	Method string `json:"method"`
}

// ListRootsResult returns available roots
type ListRootsResult struct {
	Result
	Roots []Root `json:"roots"`
}

/* Completion */

// CompleteRequest seeks argument completions
type CompleteRequest struct {
	Method string `json:"method"`
	Params struct {
		// Reference to prompt or resource
		Ref interface{} `json:"ref"`
		// Argument to complete
		Argument struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"argument"`
	} `json:"params"`
}

// CompleteResult provides completion suggestions
type CompleteResult struct {
	Result
	Completion struct {
		// Suggested values (max 100)
		Values []string `json:"values"`
		// Total available completions
		Total int `json:"total,omitempty"`
		// More completions available
		HasMore bool `json:"hasMore,omitempty"`
	} `json:"completion"`
}

// ResourceReference identifies a resource
type ResourceReference struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

// PromptReference identifies a prompt
type PromptReference struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

/* Request/Response Types */

// ListResourcesRequest queries available resources
type ListResourcesRequest struct {
	PaginatedRequest
	Method string `json:"method"`
}

// ListResourcesResult returns available resources
type ListResourcesResult struct {
	PaginatedResult
	Resources []Resource `json:"resources"`
}

// ListResourceTemplatesRequest queries resource templates
type ListResourceTemplatesRequest struct {
	PaginatedRequest
	Method string `json:"method"`
}

// ListResourceTemplatesResult returns resource templates
type ListResourceTemplatesResult struct {
	PaginatedResult
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

// ReadResourceRequest fetches resource content
type ReadResourceRequest struct {
	Method string `json:"method"`
	Params struct {
		// Resource URI
		URI string `json:"uri"`
		// Optional arguments
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	} `json:"params"`
}

// ReadResourceResult returns resource content
type ReadResourceResult struct {
	Result
	Contents []ResourceContents `json:"contents"`
}

// SubscribeRequest subscribes to resource changes
type SubscribeRequest struct {
	Method string `json:"method"`
	Params struct {
		// Resource URI to monitor
		URI string `json:"uri"`
	} `json:"params"`
}

// UnsubscribeRequest cancels resource subscription
type UnsubscribeRequest struct {
	Method string `json:"method"`
	Params struct {
		// Resource URI to stop monitoring
		URI string `json:"uri"`
	} `json:"params"`
}

// ResourceUpdatedNotification signals resource changes
type ResourceUpdatedNotification struct {
	Notification
	Params struct {
		// Changed resource URI
		URI string `json:"uri"`
	} `json:"params"`
}

// ResourceListChangedNotification signals resource list changes
type ResourceListChangedNotification struct {
	Notification
}

// ListToolsRequest queries available tools
type ListToolsRequest struct {
	PaginatedRequest
	Method string `json:"method"`
}

// ListToolsResult returns available tools
type ListToolsResult struct {
	PaginatedResult
	Tools []Tool `json:"tools"`
}

// CallToolRequest executes a tool
type CallToolRequest struct {
	Method string `json:"method"`
	Params struct {
		// Tool identifier
		Name string `json:"name"`
		// Tool arguments
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	} `json:"params"`
}

// CallToolResult returns tool execution results
type CallToolResult struct {
	Result
	// Result content
	Content []Content `json:"content"`
	// Indicates error occurred
	IsError bool `json:"isError,omitempty"`
}

// ToolListChangedNotification signals tool list changes
type ToolListChangedNotification struct {
	Notification
}

// ListPromptsRequest queries available prompts
type ListPromptsRequest struct {
	PaginatedRequest
	Method string `json:"method"`
}

// ListPromptsResult returns available prompts
type ListPromptsResult struct {
	PaginatedResult
	Prompts []Prompt `json:"prompts"`
}

// GetPromptRequest fetches a prompt
type GetPromptRequest struct {
	Method string `json:"method"`
	Params struct {
		// Prompt identifier
		Name string `json:"name"`
		// Template arguments
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	} `json:"params"`
}

// GetPromptResult returns prompt content
type GetPromptResult struct {
	Result
	// Optional prompt text
	Prompt string `json:"prompt,omitempty"`
	// Structured messages
	Messages []PromptMessage `json:"messages"`
	// Prompt description
	Description string `json:"description,omitempty"`
}

// PromptListChangedNotification signals prompt list changes
type PromptListChangedNotification struct {
	Notification
}

// RootsListChangedNotification signals root list changes
type RootsListChangedNotification struct {
	Notification
}

// PingRequest validates connection
type PingRequest struct {
	Method string `json:"method"`
}

// PingResult confirms connection
type PingResult struct {
	Result
}

/* Client Messages */

// ClientRequest represents all client-initiated requests
type ClientRequest interface {
	JSONRPCMessage
}

// ClientNotification represents all client notifications
type ClientNotification interface {
	JSONRPCMessage
}

// ClientResult represents all client results
type ClientResult interface {
	JSONRPCMessage
}

/* Server Messages */

// ServerRequest represents all server-initiated requests
type ServerRequest interface {
	JSONRPCMessage
}

// ServerNotification represents all server notifications
type ServerNotification interface {
	JSONRPCMessage
}

// ServerResult represents all server results
type ServerResult interface {
	JSONRPCMessage
}
