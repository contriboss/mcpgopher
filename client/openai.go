package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
)

func (c *HTTPClient) OpenaiTools() ([]openai.ChatCompletionToolParam, error) {
	ctx := context.Background()
	err := c.Initialize(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := c.RawRequest(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	data := map[string]interface{}{}
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return nil, err
	}

	toolsRaw := data["result"].(map[string]interface{})["tools"].([]interface{})
	tools := []openai.ChatCompletionToolParam{}
	for _, toolRaw := range toolsRaw {
		tool := openai.ChatCompletionToolParam{}
		toolMap := toolRaw.(map[string]interface{})

		normalizedTool := mcpToVendor(toolMap)
		fmt.Println(normalizedTool)
		function := normalizedTool["function"].(map[string]interface{})
		parameters := function["parameters"].(map[string]interface{})

		//inputSchema := toolMap["inputSchema"].(map[string]interface{})
		tool.Function.Name = toolMap["name"].(string)
		tool.Function.Description = param.Opt[string]{
			Value:  toolMap["description"].(string),
			Status: 2,
		}
		tool.Function.Parameters = parameters

		tools = append(tools, tool)
	}

	return tools, nil
}

// mcpToVendor converts MCP format to vendor format
func mcpToVendor(toolMap map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        toolMap["name"],
			"description": toolMap["description"],
			"parameters":  normalizeSchema(toolMap["inputSchema"].(map[string]interface{})),
		},
	}
}

// normalizeSchema normalizes the schema structure
func normalizeSchema(schema map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy all elements except those to be excluded
	for k, v := range schema {
		if k != "annotations" && k != "outputSchema" {
			result[k] = v
		}
	}

	// Handle specific schema types
	schemaType, ok := schema["type"].(string)
	if ok {
		switch schemaType {
		case "array":
			// Add default items if not present
			if _, hasItems := result["items"]; !hasItems {
				result["items"] = map[string]interface{}{
					"type": "string",
				}
			}
		case "object":
			// Process nested properties
			properties, hasProps := result["properties"].(map[string]interface{})
			if hasProps {
				for propName, propValue := range properties {
					if propValueMap, ok := propValue.(map[string]interface{}); ok {
						properties[propName] = normalizeSchema(propValueMap)
					}
				}
				result["properties"] = properties
			}
		}
	}

	return result
}
