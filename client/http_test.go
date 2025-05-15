package client

import (
	"context"
	"testing"
)

func TestHTTPClient(t *testing.T) {
	// Create client with custom protocol version
	client, err := NewHTTPClient(&Options{
		BaseURL:         "http://localhost:62770",
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

}

func TestProtocolVersionSentCorrectly(t *testing.T) {
	// Create client with no explicit protocol version
	client, err := NewHTTPClient(&Options{
		BaseURL: "http://localhost:62770",
		Debug:   true,
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
}

func TestPing(t *testing.T) {
	// Create client
	client, err := NewHTTPClient(&Options{
		BaseURL: "http://localhost:62770",
		Debug:   true,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize first
	ctx := context.Background()
	err = client.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Test Ping
	err = client.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}
