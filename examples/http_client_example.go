package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/contriboss/mcpgopher/client"
)

func main() {
	// This is just to test, the final client should auto connect
	// Create client with custom options
	mcp, err := client.NewHTTPClient(&client.Options{
		BaseURL: "https://api.cs.commonstaging.me/action_mcp", // Default to ActionMCP port
		Headers: map[string]string{
			"User-Agent": "mcpgopher-example/1.0",
		},
		Timeout: 30, // 30 seconds
		Debug:   true,
		Logger:  os.Stdout,
	})
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}
	defer mcp.Close()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Initialize the connection
	fmt.Println("Initializing connection...")
	err = mcp.Initialize(ctx)
	if err != nil {
		fmt.Printf("Failed to initialize: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Connection initialized with protocol version 2025-03-26. Session ID: %s\n", mcp.GetSessionID())

	// List available tools before ping
	fmt.Println("Listing available tools...")
	raw, err := mcp.RawRequest(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		fmt.Printf("tools/list request failed: %v\n", err)
	} else {
		fmt.Printf("tools/list full envelope: %s\n", string(raw))
	}

	// Make a ping request using the dedicated Ping method
	fmt.Println("Sending ping request...")
	startTime := time.Now()
	err = mcp.Ping(ctx)
	pingDuration := time.Since(startTime)
	if err != nil {
		fmt.Printf("Ping request failed: %v\n", err)
	} else {
		fmt.Printf("Ping successful! Response time: %s\n", pingDuration)
	}

	// Also make a ping request using the generic Request method to see full response data
	fmt.Println("Sending ping request with Request method...")
	result, err := mcp.Request(ctx, "ping", nil)
	if err != nil {
		fmt.Printf("Ping request failed: %v\n", err)
	} else {
		var prettyJSON map[string]interface{}
		json.Unmarshal(result, &prettyJSON)
		jsonStr, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Printf("Ping response: %s\n", jsonStr)
	}

	result, err = mcp.RawRequest(ctx, "tools/call", map[string]interface{}{
		"name": "identify_company",
		"arguments": map[string]interface{}{
			"company_name": "ad blue",
		},
	})

	fmt.Println(string(result))

	// Wait for Ctrl+C to exit
	fmt.Println("Press Ctrl+C to exit")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("Exiting...")
}
