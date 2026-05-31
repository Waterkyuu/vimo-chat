package mcp

import (
	"testing"

	"vimo-chat/internal/config"
)

func TestNewManagerWithEmptyConfig(t *testing.T) {
	m := NewManager(nil)
	if m == nil {
		t.Fatal("manager is nil")
	}
	if len(m.Clients()) != 0 {
		t.Fatalf("expected 0 clients, got %d", len(m.Clients()))
	}
}

func TestNewManagerCreatesClientsFromConfig(t *testing.T) {
	servers := map[string]config.MCPServerConfig{
		"test-sse": {
			Transport: config.TransportSSE,
			URL:       "http://127.0.0.1:1/sse",
		},
		"test-http": {
			Transport: config.TransportStreamableHTTP,
			URL:       "http://127.0.0.1:1/mcp",
		},
	}

	m := NewManager(servers)
	defer m.Close()

	clients := m.Clients()
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}

	if _, ok := clients["test-sse"]; !ok {
		t.Fatal("missing test-sse client")
	}
	if _, ok := clients["test-http"]; !ok {
		t.Fatal("missing test-http client")
	}
}

func TestNewManagerSkipsInvalidTransport(t *testing.T) {
	servers := map[string]config.MCPServerConfig{
		"bad": {
			Transport: "invalid",
		},
		"good": {
			Transport: config.TransportSSE,
			URL:       "http://127.0.0.1:1/sse",
		},
	}

	m := NewManager(servers)
	defer m.Close()

	clients := m.Clients()
	if len(clients) != 1 {
		t.Fatalf("expected 1 client (skipping bad), got %d", len(clients))
	}
	if _, ok := clients["good"]; !ok {
		t.Fatal("missing good client")
	}
}

func TestManagerCloseIdempotent(t *testing.T) {
	servers := map[string]config.MCPServerConfig{
		"test": {
			Transport: config.TransportSSE,
			URL:       "http://127.0.0.1:1/sse",
		},
	}

	m := NewManager(servers)

	if err := m.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}
