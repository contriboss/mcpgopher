package main

import (
	"encoding/json"
)

// Protocol version
const ProtocolVersion = "2025-06-18"

// Base message types
type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
	return e.Message
}

type Request struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

type Result struct {
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

type Notification struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// Progress and pagination
type ProgressToken string
type Cursor string

// Capabilities
type ClientCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Roots        *RootsCapability       `json:"roots,omitempty"`
	Sampling     *SamplingCapability    `json:"sampling,omitempty"`
	Elicitation  *ElicitationCapability `json:"elicitation,omitempty"`
}

type RootsCapability struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}

type SamplingCapability struct{}

type ElicitationCapability struct{}

type ServerCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Logging      *LoggingCapability     `json:"logging,omitempty"`
	Completions  *CompletionsCapability `json:"completions,omitempty"`
	Prompts      *PromptsCapability     `json:"prompts,omitempty"`
	Resources    *ResourcesCapability   `json:"resources,omitempty"`
	Tools        *ToolsCapability       `json:"tools,omitempty"`
}

type LoggingCapability struct{}
type CompletionsCapability struct{}

type PromptsCapability struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   *bool `json:"subscribe,omitempty"`
	ListChanged *bool `json:"listChanged,omitempty"`
}

type ToolsCapability struct {
	ListChanged *bool `json:"listChanged,omitempty"`
}

// Implementation info
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Title   string `json:"title,omitempty"`
}

// Initialize request/response
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ServerCapabilities     `json:"capabilities"`
	ServerInfo      Implementation         `json:"serverInfo"`
	Instructions    string                 `json:"instructions,omitempty"`
	Meta            map[string]interface{} `json:"_meta,omitempty"`
}

// Empty result for operations that don't return data
type EmptyResult struct {
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

// Ping request (no params needed)
type PingParams struct{}

// Logging
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

type SetLevelParams struct {
	Level LoggingLevel `json:"level"`
}

// Resources
type Resource struct {
	URI         string                 `json:"uri"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Meta        map[string]interface{} `json:"_meta,omitempty"`
}

type ResourceTemplate struct {
	URITemplate string                 `json:"uriTemplate"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	MimeType    string                 `json:"mimeType,omitempty"`
	Meta        map[string]interface{} `json:"_meta,omitempty"`
}

type ResourceContents struct {
	Type     string                 `json:"type"`
	URI      string                 `json:"uri,omitempty"`
	MimeType string                 `json:"mimeType,omitempty"`
	Text     string                 `json:"text,omitempty"`
	Blob     string                 `json:"blob,omitempty"`
	Meta     map[string]interface{} `json:"_meta,omitempty"`
}

type ListResourcesParams struct {
	Cursor *Cursor `json:"cursor,omitempty"`
}

type ListResourcesResult struct {
	Resources  []Resource             `json:"resources"`
	NextCursor *Cursor                `json:"nextCursor,omitempty"`
	Meta       map[string]interface{} `json:"_meta,omitempty"`
}

type ListResourceTemplatesParams struct {
	Cursor *Cursor `json:"cursor,omitempty"`
}

type ListResourceTemplatesResult struct {
	ResourceTemplates []ResourceTemplate     `json:"resourceTemplates"`
	NextCursor        *Cursor                `json:"nextCursor,omitempty"`
	Meta              map[string]interface{} `json:"_meta,omitempty"`
}

type ReadResourceParams struct {
	URI string `json:"uri"`
}

type ReadResourceResult struct {
	Contents []ResourceContents     `json:"contents"`
	Meta     map[string]interface{} `json:"_meta,omitempty"`
}

type SubscribeParams struct {
	URI string `json:"uri"`
}

type UnsubscribeParams struct {
	URI string `json:"uri"`
}

// Tools
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Meta        map[string]interface{} `json:"_meta,omitempty"`
}

type ToolContent struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	MimeType string                 `json:"mimeType,omitempty"`
	Data     string                 `json:"data,omitempty"`
	Meta     map[string]interface{} `json:"_meta,omitempty"`
}

type ListToolsParams struct {
	Cursor *Cursor `json:"cursor,omitempty"`
}

type ListToolsResult struct {
	Tools      []Tool                 `json:"tools"`
	NextCursor *Cursor                `json:"nextCursor,omitempty"`
	Meta       map[string]interface{} `json:"_meta,omitempty"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ToolContent          `json:"content"`
	IsError *bool                  `json:"isError,omitempty"`
	Meta    map[string]interface{} `json:"_meta,omitempty"`
}

// Prompts
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    *bool  `json:"required,omitempty"`
}

type Prompt struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Arguments   []PromptArgument       `json:"arguments,omitempty"`
	Meta        map[string]interface{} `json:"_meta,omitempty"`
}

type MessageContent struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	MimeType string                 `json:"mimeType,omitempty"`
	Data     string                 `json:"data,omitempty"`
	Meta     map[string]interface{} `json:"_meta,omitempty"`
}

type PromptMessage struct {
	Role    string                 `json:"role"`
	Content []MessageContent       `json:"content"`
	Meta    map[string]interface{} `json:"_meta,omitempty"`
}

type ListPromptsParams struct {
	Cursor *Cursor `json:"cursor,omitempty"`
}

type ListPromptsResult struct {
	Prompts    []Prompt               `json:"prompts"`
	NextCursor *Cursor                `json:"nextCursor,omitempty"`
	Meta       map[string]interface{} `json:"_meta,omitempty"`
}

type GetPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type GetPromptResult struct {
	Description string                 `json:"description,omitempty"`
	Messages    []PromptMessage        `json:"messages"`
	Meta        map[string]interface{} `json:"_meta,omitempty"`
}

// Completions
type CompletionRef struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type CompleteArgument struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CompleteParams struct {
	Ref      CompletionRef    `json:"ref"`
	Argument CompleteArgument `json:"argument"`
}

type Completion struct {
	Values  []string               `json:"values"`
	Total   *int                   `json:"total,omitempty"`
	HasMore *bool                  `json:"hasMore,omitempty"`
	Meta    map[string]interface{} `json:"_meta,omitempty"`
}

type CompleteResult struct {
	Completion Completion             `json:"completion"`
	Meta       map[string]interface{} `json:"_meta,omitempty"`
}

// Roots
type Root struct {
	URI  string                 `json:"uri"`
	Name string                 `json:"name,omitempty"`
	Meta map[string]interface{} `json:"_meta,omitempty"`
}

type ListRootsParams struct{}

type ListRootsResult struct {
	Roots []Root                 `json:"roots"`
	Meta  map[string]interface{} `json:"_meta,omitempty"`
}

// Sampling
type SamplingMessage struct {
	Role    string                 `json:"role"`
	Content []MessageContent       `json:"content"`
	Meta    map[string]interface{} `json:"_meta,omitempty"`
}

type ModelHint struct {
	Name string `json:"name,omitempty"`
}

type ModelPreferences struct {
	CostPriority         *float64    `json:"costPriority,omitempty"`
	SpeedPriority        *float64    `json:"speedPriority,omitempty"`
	IntelligencePriority *float64    `json:"intelligencePriority,omitempty"`
	Hints                []ModelHint `json:"hints,omitempty"`
}

type CreateMessageParams struct {
	Messages         []SamplingMessage      `json:"messages"`
	ModelPreferences *ModelPreferences      `json:"modelPreferences,omitempty"`
	SystemPrompt     string                 `json:"systemPrompt,omitempty"`
	IncludeContext   string                 `json:"includeContext,omitempty"`
	Temperature      *float64               `json:"temperature,omitempty"`
	MaxTokens        *int                   `json:"maxTokens,omitempty"`
	StopSequences    []string               `json:"stopSequences,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type CreateMessageResult struct {
	Role       string                 `json:"role"`
	Content    []MessageContent       `json:"content"`
	Model      string                 `json:"model"`
	StopReason string                 `json:"stopReason,omitempty"`
	Meta       map[string]interface{} `json:"_meta,omitempty"`
}

// Elicitation
type StringSchema struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MinLength   *int   `json:"minLength,omitempty"`
	MaxLength   *int   `json:"maxLength,omitempty"`
	Format      string `json:"format,omitempty"`
}

type NumberSchema struct {
	Type        string   `json:"type"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
}

type BooleanSchema struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Default     *bool  `json:"default,omitempty"`
}

type EnumSchema struct {
	Type        string   `json:"type"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum"`
	EnumNames   []string `json:"enumNames,omitempty"`
}

type ElicitParams struct {
	Message         string `json:"message"`
	RequestedSchema struct {
		Type       string                 `json:"type"`
		Properties map[string]interface{} `json:"properties"`
		Required   []string               `json:"required,omitempty"`
	} `json:"requestedSchema"`
}

type ElicitResult struct {
	Action  string                 `json:"action"`
	Content map[string]interface{} `json:"content,omitempty"`
}

// Progress notifications
type ProgressNotificationParams struct {
	ProgressToken ProgressToken `json:"progressToken"`
	Progress      float64       `json:"progress"`
	Total         *float64      `json:"total,omitempty"`
	Message       string        `json:"message,omitempty"`
}

// Logging notifications
type LoggingMessageNotificationParams struct {
	Level  LoggingLevel `json:"level"`
	Data   interface{}  `json:"data"`
	Logger string       `json:"logger,omitempty"`
}

// Resource notifications
type ResourceUpdatedNotificationParams struct {
	URI string `json:"uri"`
}
