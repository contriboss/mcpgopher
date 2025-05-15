package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// mockMCPServer creates a simple MCP server for testing
func mockMCPServer() *httptest.Server {
	var sessionID string
	var sessionMu sync.Mutex
	var receivedProtocolVersion string

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Get method
		method, ok := req["method"].(string)
		if !ok {
			http.Error(w, "Invalid method", http.StatusBadRequest)
			return
		}

		switch method {
		case "initialize":
			// Check for protocol version, clientInfo and capabilities
			if params, ok := req["params"].(map[string]interface{}); ok {
				if version, ok := params["protocolVersion"].(string); ok {
					// Store the received protocol version
					receivedProtocolVersion = version
				}

				// Verify clientInfo was provided
				if clientInfo, ok := params["clientInfo"].(map[string]interface{}); ok {
					if name, ok := clientInfo["name"].(string); ok && name == "mcpgopher" {
						// Client name is correct
					} else {
						http.Error(w, "Invalid clientInfo name", http.StatusBadRequest)
						return
					}
				} else {
					http.Error(w, "Missing or invalid 'clientInfo'", http.StatusBadRequest)
					return
				}
			}

			// Generate session ID
			sessionMu.Lock()
			sessionID = "test-session-" + time.Now().String()
			sessionMu.Unlock()

			// Set session ID header
			w.Header().Set("Mcp-Session-Id", sessionID)
			w.Header().Set("Content-Type", "application/json")

			// Return success with protocol version
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"status":          "initialized",
					"protocolVersion": receivedProtocolVersion,
					"clientInfo": map[string]interface{}{
						"name":    "mcpgopher",
						"version": "0.1.0",
					},
				},
			}); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}



		case "error":
			// Return error response
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"error": map[string]interface{}{
					"code":    -32000,
					"message": "Test error",
				},
			}); err != nil {
				http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
				return
			}

		case "input":
			// This is a notification, no response needed
			w.WriteHeader(http.StatusOK)
		}
	}))
}

func TestHTTPClient(t *testing.T) {
	// Start mock server
	server := mockMCPServer()
	defer server.Close()

	// Create client with custom protocol version
	client, err := NewHTTPClient(&Options{
		BaseURL:         server.URL,
		Debug:           true,
		ProtocolVersion: "2025-03-26",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test Initialize
	ctx := context.Background()
	err = client.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test GetSessionID
	sessionID := client.GetSessionID()
	if sessionID == "" {
		t.Fatalf("Expected non-empty session ID")
	}





	// Test request with error response
	_, err = client.Request(ctx, "error", nil)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestNotifications(t *testing.T) {
	// Create a simple notification handler test
	notificationChan := make(chan string, 1)
	notificationHandler := func(method string, params map[string]interface{}) {
		notificationChan <- method
	}

	// Create client with the handler
	client, err := NewHTTPClient(&Options{
		BaseURL:         "http://localhost:8080", // URL doesn't matter for this test
		ProtocolVersion: "2025-03-26",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Set notification handler
	client.SetNotificationHandler(notificationHandler)

	// Verify handler was set
	if client.notificationHandler == nil {
		t.Fatalf("Notification handler was not set")
	}
}

func TestProtocolVersionDefault(t *testing.T) {
	// Start mock server
	server := mockMCPServer()
	defer server.Close()

	// Create client with no explicit protocol version
	client, err := NewHTTPClient(&Options{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test Initialize
	ctx := context.Background()
	err = client.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}



	result, err := client.Request(ctx, "initialize", nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify default protocol version was sent correctly
	if protocolVersion, ok := resultMap["protocolVersion"].(string); !ok || protocolVersion != "2025-03-26" {
		t.Errorf("Expected default protocol version '2025-03-26', got %v", protocolVersion)
	}
}
