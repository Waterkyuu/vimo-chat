package mcp

import (
	"fmt"

	"vimo-chat/internal/config"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

func NewClient(cfg config.MCPServerConfig) (*client.Client, error) {
	switch cfg.Transport {
	case config.TransportStdio:
		return newStdioClient(cfg)
	case config.TransportSSE:
		return newSSEClient(cfg)
	case config.TransportStreamableHTTP:
		return newStreamableHTTPClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported transport: %s", cfg.Transport)
	}
}

func newStdioClient(cfg config.MCPServerConfig) (*client.Client, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("stdio transport requires a command")
	}

	env := make([]string, 0, len(cfg.Env))
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}

	t := transport.NewStdioWithOptions(cfg.Command, env, cfg.Args)
	return client.NewClient(t), nil
}

func newSSEClient(cfg config.MCPServerConfig) (*client.Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("SSE transport requires a URL")
	}

	opts := make([]transport.ClientOption, 0, len(cfg.Headers))
	if len(cfg.Headers) > 0 {
		opts = append(opts, transport.WithHeaders(cfg.Headers))
	}

	t, err := transport.NewSSE(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE transport: %w", err)
	}

	return client.NewClient(t), nil
}

func newStreamableHTTPClient(cfg config.MCPServerConfig) (*client.Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("streamable HTTP transport requires a URL")
	}

	opts := make([]transport.StreamableHTTPCOption, 0, len(cfg.Headers))
	if len(cfg.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(cfg.Headers))
	}

	t, err := transport.NewStreamableHTTP(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamable HTTP transport: %w", err)
	}

	var clientOpts []client.ClientOption
	if sid := t.GetSessionId(); sid != "" {
		clientOpts = append(clientOpts, client.WithSession())
	}

	return client.NewClient(t, clientOpts...), nil
}
