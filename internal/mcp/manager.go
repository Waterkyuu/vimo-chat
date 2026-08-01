package mcp

import (
	"context"
	"fmt"
	"log"
	"strings"
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

// StartAndInit starts every client and performs the MCP handshake. Each server
// is initialized independently: a failure is logged, the failing client is
// dropped, and the remaining servers keep working. ctx bounds the startup time.
// A non-nil error lists the servers that failed; healthy clients are still
// available via LoadTools.
func (m *Manager) StartAndInit(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	failed := make(map[string]*client.Client)
	var errs []string

	for name, c := range m.clients {
		if err := c.Start(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("start %q: %v", name, err))
			failed[name] = c
			continue
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
			errs = append(errs, fmt.Sprintf("initialize %q: %v", name, err))
			failed[name] = c
			continue
		}
	}

	for name, c := range failed {
		_ = c.Close()
		delete(m.clients, name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("mcp startup completed with errors: %s", strings.Join(errs, "; "))
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
