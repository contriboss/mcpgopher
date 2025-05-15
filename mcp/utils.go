package mcp

import (
	"encoding/json"
	"fmt"
)

// ParseCallToolResult parses a raw JSON message into a CallToolResult.
func ParseCallToolResult(rawMessage *json.RawMessage) (*CallToolResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result CallToolResult

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = metaMap
		}
	}

	isError, ok := jsonContent["isError"]
	if ok {
		if isErrorBool, ok := isError.(bool); ok {
			result.IsError = isErrorBool
		}
	}

	contents, ok := jsonContent["content"]
	if !ok {
		return nil, fmt.Errorf("content is missing")
	}

	contentArr, ok := contents.([]any)
	if !ok {
		return nil, fmt.Errorf("content is not an array")
	}

	for _, content := range contentArr {
		// Extract content
		contentMap, ok := content.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content is not an object")
		}

		// Process content
		content, err := ParseContent(contentMap)
		if err != nil {
			return nil, err
		}

		result.Content = append(result.Content, content)
	}

	return &result, nil
}

// ParseReadResourceResult parses a raw JSON message into a ReadResourceResult.
func ParseReadResourceResult(rawMessage *json.RawMessage) (*ReadResourceResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result ReadResourceResult

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = metaMap
		}
	}

	contents, ok := jsonContent["contents"]
	if !ok {
		return nil, fmt.Errorf("contents is missing")
	}

	contentArr, ok := contents.([]any)
	if !ok {
		return nil, fmt.Errorf("contents is not an array")
	}

	for _, content := range contentArr {
		// Extract content
		contentMap, ok := content.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content is not an object")
		}

		// Process content
		content, err := ParseResourceContents(contentMap)
		if err != nil {
			return nil, err
		}

		result.Contents = append(result.Contents, content)
	}

	return &result, nil
}

// ParseGetPromptResult parses a raw JSON message into a GetPromptResult.
func ParseGetPromptResult(rawMessage *json.RawMessage) (*GetPromptResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	result := GetPromptResult{}

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = metaMap
		}
	}

	description, ok := jsonContent["description"]
	if ok {
		if descriptionStr, ok := description.(string); ok {
			result.Description = descriptionStr
		}
	}

	messages, ok := jsonContent["messages"]
	if ok {
		messagesArr, ok := messages.([]any)
		if !ok {
			return nil, fmt.Errorf("messages is not an array")
		}

		for _, message := range messagesArr {
			messageMap, ok := message.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("message is not an object")
			}

			// Extract role
			roleStr := ExtractString(messageMap, "role")
			if roleStr == "" || (roleStr != string(RoleAssistant) && roleStr != string(RoleUser)) {
				return nil, fmt.Errorf("unsupported role: %s", roleStr)
			}

			// Extract content
			contentMap, ok := messageMap["content"].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("content is not an object")
			}

			// Process content
			content, err := ParseContent(contentMap)
			if err != nil {
				return nil, err
			}

			// Append processed message
			result.Messages = append(result.Messages, NewPromptMessage(Role(roleStr), content))
		}
	}

	return &result, nil
}

// ParseContent parses a content map into a Content interface.
func ParseContent(contentMap map[string]any) (Content, error) {
	contentType := ExtractString(contentMap, "type")

	switch contentType {
	case "text":
		text := ExtractString(contentMap, "text")
		return NewTextContent(text), nil

	case "image":
		data := ExtractString(contentMap, "data")
		mimeType := ExtractString(contentMap, "mimeType")
		if data == "" || mimeType == "" {
			return nil, fmt.Errorf("image data or mimeType is missing")
		}
		return NewImageContent(data, mimeType), nil

	case "audio":
		data := ExtractString(contentMap, "data")
		mimeType := ExtractString(contentMap, "mimeType")
		if data == "" || mimeType == "" {
			return nil, fmt.Errorf("audio data or mimeType is missing")
		}
		return NewAudioContent(data, mimeType), nil

	case "resource":
		resourceMap := ExtractMap(contentMap, "resource")
		if resourceMap == nil {
			return nil, fmt.Errorf("resource is missing")
		}

		resourceContents, err := ParseResourceContents(resourceMap)
		if err != nil {
			return nil, err
		}

		return NewEmbeddedResource(resourceContents), nil
	}

	return nil, fmt.Errorf("unsupported content type: %s", contentType)
}

// ParseResourceContents parses a resource contents map into a ResourceContents interface.
func ParseResourceContents(contentMap map[string]any) (ResourceContents, error) {
	uri := ExtractString(contentMap, "uri")
	if uri == "" {
		return nil, fmt.Errorf("resource uri is missing")
	}

	mimeType := ExtractString(contentMap, "mimeType")

	if text := ExtractString(contentMap, "text"); text != "" {
		return TextResourceContents{
			URI:      uri,
			MimeType: mimeType,
			Text:     text,
		}, nil
	}

	if blob := ExtractString(contentMap, "blob"); blob != "" {
		return BlobResourceContents{
			URI:      uri,
			MimeType: mimeType,
			Blob:     blob,
		}, nil
	}

	return nil, fmt.Errorf("unsupported resource type")
}

// ExtractString extracts a string value from a map.
func ExtractString(data map[string]any, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// ExtractMap extracts a map from a map.
func ExtractMap(data map[string]any, key string) map[string]any {
	if value, ok := data[key]; ok {
		if m, ok := value.(map[string]any); ok {
			return m
		}
	}
	return nil
}

// NewTextContent creates a new TextContent with the given text.
func NewTextContent(text string) TextContent {
	return TextContent{
		Type: "text",
		Text: text,
	}
}

// NewImageContent creates a new ImageContent with the given data and MIME type.
func NewImageContent(data, mimeType string) ImageContent {
	return ImageContent{
		Type:     "image",
		Data:     data,
		MimeType: mimeType,
	}
}

// NewAudioContent creates a new AudioContent with the given data and MIME type.
func NewAudioContent(data, mimeType string) AudioContent {
	return AudioContent{
		Type:     "audio",
		Data:     data,
		MimeType: mimeType,
	}
}

// NewPromptMessage creates a new PromptMessage with the given role and content.
func NewPromptMessage(role Role, content Content) PromptMessage {
	return PromptMessage{
		Role:    role,
		Content: content,
	}
}

// NewEmbeddedResource creates a new EmbeddedResource with the given resource.
func NewEmbeddedResource(resource ResourceContents) EmbeddedResource {
	return EmbeddedResource{
		Type:     "resource",
		Resource: resource,
	}
}

// NewToolResultText creates a new CallToolResult with text content.
func NewToolResultText(text string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

// ToBoolPtr returns a pointer to the given boolean value.
func ToBoolPtr(b bool) *bool {
	return &b
}