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
		BaseURL: "http://localhost:62770", // Default to ActionMCP port
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

	// Make a ping request (supported by MCP spec)
	fmt.Println("Sending ping request...")
	result, err := mcp.Request(ctx, "ping", nil)
	if err != nil {
		fmt.Printf("Ping request failed: %v\n", err)
	} else {
		var prettyJSON map[string]interface{}
		json.Unmarshal(result, &prettyJSON)
		jsonStr, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Printf("Ping response: %s\n", jsonStr)
	}

	// Wait for Ctrl+C to exit
	fmt.Println("Press Ctrl+C to exit")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("Exiting...")
}
