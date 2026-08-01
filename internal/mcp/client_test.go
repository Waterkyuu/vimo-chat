package mcp

import (
	"testing"

	"vimo-chat/internal/config"
)

func TestNewClientRejectsUnknownTransport(t *testing.T) {
	_, err := NewClient(config.MCPServerConfig{
		Transport: "unknown",
	})
	if err == nil {
		t.Fatal("expected error for unknown transport, got nil")
	}
}

func TestNewClientRejectsEmptyURLForSSE(t *testing.T) {
	_, err := NewClient(config.MCPServerConfig{
		Transport: config.TransportSSE,
	})
	if err == nil {
		t.Fatal("expected error for SSE with empty URL, got nil")
	}
}

func TestNewClientRejectsEmptyURLForStreamableHTTP(t *testing.T) {
	_, err := NewClient(config.MCPServerConfig{
		Transport: config.TransportStreamableHTTP,
	})
	if err == nil {
		t.Fatal("expected error for streamable_http with empty URL, got nil")
	}
}

func TestNewClientRejectsEmptyCommandForStdio(t *testing.T) {
	_, err := NewClient(config.MCPServerConfig{
		Transport: config.TransportStdio,
	})
	if err == nil {
		t.Fatal("expected error for stdio with empty command, got nil")
	}
}

func TestNewClientCreatesSSEClient(t *testing.T) {
	c, err := NewClient(config.MCPServerConfig{
		Transport: config.TransportSSE,
		URL:       "http://127.0.0.1:9999/sse",
		Headers:   map[string]string{"X-Test": "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	c.Close()
}

func TestNewClientCreatesStreamableHTTPClient(t *testing.T) {
	c, err := NewClient(config.MCPServerConfig{
		Transport: config.TransportStreamableHTTP,
		URL:       "http://127.0.0.1:9999/mcp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	c.Close()
}

func TestNewClientCreatesStdioClientWithEcho(t *testing.T) {
	c, err := NewClient(config.MCPServerConfig{
		Transport: config.TransportStdio,
		Command:   "echo",
		Args:      []string{"hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
	c.Close()
}
