package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"

	"vimo-chat/internal/config"

	mcpTool "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type Manager struct {
	mu      sync.Mutex
	clients map[string]*client.Client
}

func NewManager(servers map[string]config.MCPServerConfig) *Manager {
	m := &Manager{
		clients: make(map[string]*client.Client, len(servers)),
	}

	for name, cfg := range servers {
		c, err := NewClient(cfg)
		if err != nil {
			log.Printf("mcp: skip server %q: %v", name, err)
			continue
		}
		m.clients[name] = c
	}

	return m
}

func (m *Manager) Clients() map[string]*client.Client {
	// return m.clients
	m.mu.Lock()
	defer m.mu.Unlock()

	clients := make(map[string]*client.Client, len(m.clients))
	for name, c := range m.clients {
		clients[name] = c
	}
	return clients
}

func (m *Manager) StartAndInit(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, c := range m.clients {
		if err := c.Start(ctx); err != nil {
			return fmt.Errorf("start %q: %w", name, err)
		}

		initReq := mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				ClientInfo: mcp.Implementation{
					Name:    "vimo-chat",
					Version: "0.1.0",
				},
			},
		}

		if _, err := c.Initialize(ctx, initReq); err != nil {
			return fmt.Errorf("initialize %q: %w", name, err)
		}
	}

	return nil
}

func (m *Manager) LoadTools(ctx context.Context) []tool.BaseTool {
	m.mu.Lock()
	clients := make(map[string]*client.Client, len(m.clients))
	for k, v := range m.clients {
		clients[k] = v
	}
	m.mu.Unlock()

	var allTools []tool.BaseTool
	for name, c := range clients {
		tools, err := mcpTool.GetTools(ctx, &mcpTool.Config{Cli: c})
		if err != nil {
			log.Printf("mcp: load tools from %q: %v", name, err)
			continue
		}
		allTools = append(allTools, tools...)
	}

	return allTools
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for name, c := range m.clients {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close %q: %w", name, err)
		}
		delete(m.clients, name)
	}
	return firstErr
}
