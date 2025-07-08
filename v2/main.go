package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	// Create client
	client := NewMCPClient("http://127.0.0.1:62770")

	// Set timeout
	client.SetTimeout(60 * time.Second)

	// Initialize connection
	fmt.Println("Initializing MCP client...")
	if err := client.Initialize(); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	fmt.Println("✓ MCP client initialized successfully")

	// Get server info
	serverInfo := client.GetServerInfo()
	if serverInfo != nil {
		fmt.Printf("Connected to: %s v%s\n", serverInfo.Name, serverInfo.Version)
		if serverInfo.Title != "" {
			fmt.Printf("Title: %s\n", serverInfo.Title)
		}
	}

	// Test ping
	fmt.Println("\n=== Testing Ping ===")
	if err := client.Ping(); err != nil {
		log.Printf("Ping failed: %v", err)
	} else {
		fmt.Println("✓ Ping successful")
	}

	// List and read resources
	if client.ServerSupportsResources() {
		fmt.Println("\n=== Resources ===")
		resources, err := client.ListResources(nil)
		if err != nil {
			log.Printf("Failed to list resources: %v", err)
		} else {
			fmt.Printf("Found %d resources:\n", len(resources.Resources))
			for _, resource := range resources.Resources {
				fmt.Printf("  - %s (%s)\n", resource.Name, resource.URI)
				if resource.Description != "" {
					fmt.Printf("    Description: %s\n", resource.Description)
				}

				// Read the resource
				content, err := client.ReadResource(resource.URI)
				if err != nil {
					log.Printf("    Failed to read resource: %v", err)
				} else {
					fmt.Printf("    Content: %d parts\n", len(content.Contents))
					for i, part := range content.Contents {
						fmt.Printf("      [%d] Type: %s", i, part.Type)
						if part.Text != "" {
							preview := part.Text
							if len(preview) > 100 {
								preview = preview[:100] + "..."
							}
							fmt.Printf(", Text: %s", preview)
						}
						fmt.Println()
					}
				}
			}
		}
	}

	// List and call tools
	if client.ServerSupportsTools() {
		fmt.Println("\n=== Tools ===")
		tools, err := client.ListTools(nil)
		if err != nil {
			log.Printf("Failed to list tools: %v", err)
		} else {
			fmt.Printf("Found %d tools:\n", len(tools.Tools))
			for _, tool := range tools.Tools {
				fmt.Printf("  - %s\n", tool.Name)
				if tool.Description != "" {
					fmt.Printf("    Description: %s\n", tool.Description)
				}

				// Try to call the tool with empty arguments
				result, err := client.CallTool(tool.Name, map[string]interface{}{})
				if err != nil {
					log.Printf("    Failed to call tool: %v", err)
				} else {
					fmt.Printf("    Result: %d content parts\n", len(result.Content))
					if result.IsError != nil && *result.IsError {
						fmt.Printf("    ⚠ Tool returned error\n")
					} else {
						fmt.Printf("    ✓ Tool executed successfully\n")
					}
				}
			}
		}
	}

	// List and get prompts
	if client.ServerSupportsPrompts() {
		fmt.Println("\n=== Prompts ===")
		prompts, err := client.ListPrompts(nil)
		if err != nil {
			log.Printf("Failed to list prompts: %v", err)
		} else {
			fmt.Printf("Found %d prompts:\n", len(prompts.Prompts))
			for _, prompt := range prompts.Prompts {
				fmt.Printf("  - %s\n", prompt.Name)
				if prompt.Description != "" {
					fmt.Printf("    Description: %s\n", prompt.Description)
				}

				// Show arguments
				if len(prompt.Arguments) > 0 {
					fmt.Printf("    Arguments:\n")
					for _, arg := range prompt.Arguments {
						required := ""
						if arg.Required != nil && *arg.Required {
							required = " (required)"
						}
						fmt.Printf("      - %s%s\n", arg.Name, required)
						if arg.Description != "" {
							fmt.Printf("        %s\n", arg.Description)
						}
					}
				}

				// Try to get the prompt with empty arguments
				result, err := client.GetPrompt(prompt.Name, map[string]string{})
				if err != nil {
					log.Printf("    Failed to get prompt: %v", err)
				} else {
					fmt.Printf("    Messages: %d\n", len(result.Messages))
					for i, msg := range result.Messages {
						fmt.Printf("      [%d] Role: %s, Content parts: %d\n", i, msg.Role, len(msg.Content))
					}
				}
			}
		}
	}

	// Test completions
	if client.ServerSupportsCompletions() {
		fmt.Println("\n=== Completions ===")
		ref := CompletionRef{
			Type: "prompt",
			Name: "test-prompt",
		}
		arg := CompleteArgument{
			Name:  "input",
			Value: "hel",
		}

		completions, err := client.Complete(ref, arg)
		if err != nil {
			log.Printf("Failed to get completions: %v", err)
		} else {
			fmt.Printf("Found %d completions:\n", len(completions.Completion.Values))
			for _, value := range completions.Completion.Values {
				fmt.Printf("  - %s\n", value)
			}
		}
	}

	// Test sampling (LLM integration)
	fmt.Println("\n=== Sampling ===")
	messages := []SamplingMessage{
		{
			Role: "user",
			Content: []MessageContent{
				{
					Type: "text",
					Text: "Hello, how are you?",
				},
			},
		},
	}

	preferences := &ModelPreferences{
		IntelligencePriority: float64Ptr(0.8),
		SpeedPriority:        float64Ptr(0.6),
		CostPriority:         float64Ptr(0.4),
		Hints: []ModelHint{
			{Name: "claude-3-5-sonnet"},
			{Name: "gpt-4"},
		},
	}

	response, err := client.CreateMessage(messages, preferences)
	if err != nil {
		log.Printf("Failed to create message: %v", err)
	} else {
		fmt.Printf("Response from %s:\n", response.Model)
		for _, content := range response.Content {
			if content.Type == "text" {
				fmt.Printf("  %s\n", content.Text)
			}
		}
		if response.StopReason != "" {
			fmt.Printf("Stop reason: %s\n", response.StopReason)
		}
	}

	// Test roots
	fmt.Println("\n=== Roots ===")
	roots, err := client.ListRoots()
	if err != nil {
		log.Printf("Failed to list roots: %v", err)
	} else {
		fmt.Printf("Found %d roots:\n", len(roots.Roots))
		for _, root := range roots.Roots {
			fmt.Printf("  - %s", root.URI)
			if root.Name != "" {
				fmt.Printf(" (%s)", root.Name)
			}
			fmt.Println()
		}
	}

	// Set logging level
	if client.ServerSupportsLogging() {
		fmt.Println("\n=== Logging ===")
		if err := client.SetLoggingLevel(LoggingLevelInfo); err != nil {
			log.Printf("Failed to set logging level: %v", err)
		} else {
			fmt.Println("✓ Logging level set to INFO")
		}
	}

	// Test batch operations
	fmt.Println("\n=== Batch Operations ===")
	batchResults, err := client.BatchListAll()
	if err != nil {
		log.Printf("Failed to run batch operations: %v", err)
	} else {
		fmt.Printf("Batch results collected: %d categories\n", len(batchResults))
		for category, data := range batchResults {
			fmt.Printf("  %s: %T\n", category, data)
		}
	}

	// Test retry logic
	fmt.Println("\n=== Retry Logic ===")
	err = client.RetryRequest(func() error {
		return client.Ping()
	}, 3)

	if err != nil {
		log.Printf("Retry operation failed: %v", err)
	} else {
		fmt.Println("✓ Retry operation succeeded")
	}

	fmt.Println("\n=== Client Demo Complete ===")
}
