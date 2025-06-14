package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// startMockStreamableHTTPServer starts a test HTTP server that implements
// a minimal Streamable HTTP server for testing purposes.
// It returns the server URL and a function to close the server.
func startMockStreamableHTTPServer() (string, func()) {
	var sessionID string
	var mu sync.Mutex

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle only POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Set content type for all responses
		w.Header().Set("Content-Type", "application/json")

		// Parse incoming JSON-RPC request
		var request map[string]any
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		method := request["method"]
		switch method {
		case "initialize":
			// Generate a new session ID
			mu.Lock()
			sessionID = fmt.Sprintf("test-session-%d", time.Now().UnixNano())
			mu.Unlock()
			w.Header().Set("Mcp-Session-Id", sessionID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			if err := json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  "initialized",
			}); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}

		case "ping":
			// Check session ID
			if r.Header.Get("Mcp-Session-Id") != sessionID {
				http.Error(w, "Invalid session ID", http.StatusNotFound)
				return
			}
			// Respond to ping with echo of params
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  request,
			}); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		case "ping_error":
			// Check session ID
			if r.Header.Get("Mcp-Session-Id") != sessionID {
				http.Error(w, "Invalid session ID", http.StatusNotFound)
				return
			}
			// Return an error response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(request)
			if err := json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"error": map[string]any{
					"code":    -1,
					"message": string(data),
				},
			}); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		}
	})

	// Start test server
	testServer := httptest.NewServer(handler)
	return testServer.URL, testServer.Close
}

func TestStreamableHTTP(t *testing.T) {
	// Start mock server
	url, closeF := startMockStreamableHTTPServer()
	defer closeF()

	// Create transport
	trans, err := NewStreamableHTTP(url)
	if err != nil {
		t.Fatal(err)
	}
	defer trans.Close()

	// Initialize the transport first
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initRequest := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "initialize",
	}

	_, err = trans.SendRequest(ctx, initRequest)
	if err != nil {
		t.Fatal(err)
	}

	// Now run the tests
	t.Run("SendRequest", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		request := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "1",
			Method:  "ping",
			Params: map[string]any{
				"string": "hello world",
				"array":  []any{1, 2, 3},
			},
		}

		// Send the request
		response, err := trans.SendRequest(ctx, request)
		if err != nil {
			t.Fatalf("SendRequest failed: %v", err)
		}

		// Parse the result to verify echo
		var result struct {
			JSONRPC string         `json:"jsonrpc"`
			ID      string         `json:"id"`
			Method  string         `json:"method"`
			Params  map[string]any `json:"params"`
		}

		if err := json.Unmarshal(response.Result, &result); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		// Verify response data matches what was sent
		if result.JSONRPC != "2.0" {
			t.Errorf("Expected JSONRPC value '2.0', got '%s'", result.JSONRPC)
		}
		if result.ID != "1" {
			t.Errorf("Expected ID '1', got '%s'", result.ID)
		}
		if result.Method != "ping" {
			t.Errorf("Expected method 'ping', got '%s'", result.Method)
		}

		if str, ok := result.Params["string"].(string); !ok || str != "hello world" {
			t.Errorf("Expected string 'hello world', got %v", result.Params["string"])
		}

		if arr, ok := result.Params["array"].([]any); !ok || len(arr) != 3 {
			t.Errorf("Expected array with 3 items, got %v", result.Params["array"])
		}
	})

	t.Run("SendRequestWithTimeout", func(t *testing.T) {
		// Create a context that's already canceled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context immediately

		// Prepare a request
		request := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "2",
			Method:  "ping",
			Params:  map[string]any{"string": "timeout"},
		}

		// The request should fail because the context is canceled
		_, err := trans.SendRequest(ctx, request)
		if err == nil {
			t.Errorf("Expected context canceled error, got nil")
		} else if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
	})

	t.Run("SendNotification & NotificationHandler", func(t *testing.T) {
		notificationChan := make(chan JSONRPCNotification, 1)

		// Set notification handler
		trans.SetNotificationHandler(func(notification JSONRPCNotification) {
			notificationChan <- notification
		})

		// Expected notification data
		expectedID := "42"
		expectedJSONRPC := "2.0"
		expectedMethod := "notifications/test"

		// Create a notification payload
		notification := JSONRPCNotification{
			JSONRPC: expectedJSONRPC,
			Method:  expectedMethod,
			Params: struct {
				AdditionalFields map[string]interface{} `json:"-"`
			}{
				AdditionalFields: map[string]interface{}{
					"id":      expectedID,
					"jsonrpc": expectedJSONRPC,
					"method":  expectedMethod,
					"foo":     "bar",
				},
			},
		}

		// Send notification to handler
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			// Wait briefly to ensure handler is ready
			time.Sleep(100 * time.Millisecond)
			trans.notificationHandler(notification)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case notification := <-notificationChan:
				// We received a notification
				got := notification.Params.AdditionalFields
				if got == nil {
					t.Errorf("Notification handler did not send the expected notification: got nil")
				}
				if got["id"] != expectedID ||
					got["jsonrpc"] != expectedJSONRPC ||
					got["method"] != expectedMethod {

					responseJson, _ := json.Marshal(got)
					expectedJson, _ := json.Marshal(map[string]string{
						"id":      expectedID,
						"jsonrpc": expectedJSONRPC,
						"method":  expectedMethod,
					})
					t.Errorf("Notification handler did not send the expected notification: \ngot %s\nexpect %s", responseJson, expectedJson)
				}

			case <-time.After(1 * time.Second):
				t.Errorf("Expected notification, got none")
			}
		}()

		wg.Wait()
	})

	t.Run("MultipleRequests", func(t *testing.T) {
		var wg sync.WaitGroup
		const numRequests = 5

		// Send multiple requests concurrently
		mu := sync.Mutex{}
		responses := make([]*JSONRPCResponse, numRequests)
		errors := make([]error, numRequests)

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Each request has a unique ID and payload
				request := JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      fmt.Sprintf("%d", 100+idx),
					Method:  "ping",
					Params:  map[string]any{"requestIndex": idx},
				}

				resp, err := trans.SendRequest(ctx, request)
				mu.Lock()
				responses[idx] = resp
				errors[idx] = err
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Check results
		for i := 0; i < numRequests; i++ {
			if errors[i] != nil {
				t.Errorf("Request %d failed: %v", i, errors[i])
				continue
			}

			// Parse the result to verify echo
			var result struct {
				JSONRPC string         `json:"jsonrpc"`
				ID      string         `json:"id"`
				Method  string         `json:"method"`
				Params  map[string]any `json:"params"`
			}

			if err := json.Unmarshal(responses[i].Result, &result); err != nil {
				t.Errorf("Request %d: Failed to unmarshal result: %v", i, err)
				continue
			}

			// Verify data matches what was sent
			expectedID := fmt.Sprintf("%d", 100+i)
			if fmt.Sprintf("%v", result.ID) != expectedID {
				t.Errorf("Request %d: Expected echoed ID %s, got %v", i, expectedID, result.ID)
			}

			if result.Method != "ping" {
				t.Errorf("Request %d: Expected method 'ping', got '%s'", i, result.Method)
			}

			// Verify the requestIndex parameter
			if idx, ok := result.Params["requestIndex"].(float64); !ok || int(idx) != i {
				t.Errorf("Request %d: Expected requestIndex %d, got %v", i, i, result.Params["requestIndex"])
			}
		}
	})

	t.Run("ResponseError", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Prepare a request
		requestID := "999"
		request := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      requestID,
			Method:  "ping_error",
			Params:  map[string]any{"foo": "bar"},
		}

		reps, err := trans.SendRequest(ctx, request)
		if err != nil {
			t.Errorf("SendRequest failed: %v", err)
		}

		if reps.Error == nil {
			t.Errorf("Expected error, got nil")
		}

		var responseError JSONRPCRequest
		if err := json.Unmarshal([]byte(reps.Error.Message), &responseError); err != nil {
			t.Errorf("Failed to unmarshal result: %v", err)
			return
		}

		if responseError.ID != requestID {
			t.Errorf("Expected ID '%s', got '%s'", requestID, responseError.ID)
		}
		if responseError.JSONRPC != "2.0" {
			t.Errorf("Expected JSONRPC '2.0', got '%s'", responseError.JSONRPC)
		}
	})
}

func TestStreamableHTTPErrors(t *testing.T) {
	t.Run("InvalidURL", func(t *testing.T) {
		// Create a new StreamableHTTP transport with an invalid URL
		_, err := NewStreamableHTTP("://invalid-url")
		if err == nil {
			t.Errorf("Expected error when creating with invalid URL, got nil")
		}
	})

	t.Run("NonExistentURL", func(t *testing.T) {
		// Create a new StreamableHTTP transport with a non-existent URL
		trans, err := NewStreamableHTTP("http://localhost:1")
		if err != nil {
			t.Fatalf("Failed to create StreamableHTTP transport: %v", err)
		}

		// Send request should fail
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		request := JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "1",
			Method:  "initialize",
		}

		_, err = trans.SendRequest(ctx, request)
		if err == nil {
			t.Errorf("Expected error when sending request to non-existent URL, got nil")
		}
	})
}
