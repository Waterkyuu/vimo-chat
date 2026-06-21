package tui

import (
	"context"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"vimo-chat/internal/chat"
	appconfig "vimo-chat/internal/config"
	"vimo-chat/internal/mcp"
	"vimo-chat/internal/memory"
)

// Message type
type chatEventMsg struct {
	event chat.Event
}

type streamDoneMsg struct{}

type errMsg struct {
	err error
}

// streamStartedMsg is emitted once the chat stream has been successfully opened.
// It carries the live event channel and the memory service (which may have just
// been created) so the model can cache them.
type streamStartedMsg struct {
	events <-chan chat.Event
	memSvc *memory.Service
}

// memoryDoneMsg signals that background memory processing finished.
type memoryDoneMsg struct {
	err error
}

// toolLoadMsg is reserved for future async tool loading (e.g. MCP).
type toolLoadMsg struct {
	tools []tool.BaseTool
	err   error
}

type inputMode int

const (
	modeNormal inputMode = iota
	modeCommandPalette
	modeProviderMenu
	modeModelMenu
	modeKeyInput
)

// Data model
type Model struct {
	// UI comp
	viewport viewport.Model
	textarea textarea.Model
	// Data
	messages []*schema.Message
	err      error
	notice   string
	// Status mark
	streaming    bool
	ready        bool
	showSplash   bool
	windowWidth  int
	windowHeight int
	// Config
	config      appconfig.Config
	configPath  string
	mode        inputMode
	menuIndex   int
	keyProvider appconfig.Provider
	keyInput    string

	// Cache the list of tools to avoid reloading local tools every time you chat
	tools []tool.BaseTool

	// Chat/memory runtime state.
	// memSvc is lazily created on the first sent message and reused afterwards.
	memSvc *memory.Service
	// mcpManager is started once at application startup and reused for the
	// lifetime of the app; its tools are merged into the chat toolset.
	mcpManager *mcp.Manager
	// streamEvents holds the active chat event channel while streaming.
	streamEvents <-chan chat.Event
	// assistantText accumulates streamed assistant deltas for live rendering.
	assistantText string
	// convID identifies the current conversation for memory extraction/summary.
	convID string
}

func NewModel() Model {
	path, err := appconfig.DefaultPath()
	if err != nil {
		path = ""
	}
	return NewModelWithConfigPath(path)
}

func NewModelWithConfigPath(configPath string) Model {
	ta := textarea.New()
	ta.Placeholder = "Send a message... (Enter to send)"
	ta.ShowLineNumbers = false
	ta.Focus()

	ta.Prompt = "| "
	ta.CharLimit = 10000

	cfg := appconfig.Default()
	var err error
	if configPath != "" {
		cfg, err = appconfig.Load(configPath)
	}

	m := Model{
		textarea:     ta,
		messages:     []*schema.Message{},
		err:          err,
		showSplash:   true,
		windowWidth:  80,
		windowHeight: 24,
		config:       cfg,
		configPath:   configPath,
		keyProvider:  cfg.ActiveProvider,
	}
	// MCP servers are started once at startup. Failures are logged but never
	// fatal: healthy servers still contribute their tools to the chat.
	m.initMCP(cfg.MCPServers)
	return m
}

// mcpStartTimeout bounds how long the application is willing to block while
// bringing MCP servers up at startup.
const mcpStartTimeout = 10 * time.Second

// initMCP builds the MCP manager from configuration and starts it. A failed or
// slow server never prevents the application from running: each server is
// started independently and only the healthy ones are kept.
func (m *Model) initMCP(servers map[string]appconfig.MCPServerConfig) {
	manager := mcp.NewManager(servers)

	ctx, cancel := context.WithTimeout(context.Background(), mcpStartTimeout)
	defer cancel()
	if err := manager.StartAndInit(ctx); err != nil {
		log.Printf("mcp startup: %v", err)
	}

	m.mcpManager = manager
}

// Close releases long-lived resources such as MCP clients. It is safe to call
// on a zero-value model and should be invoked once when the application exits.
func (m Model) Close() error {
	if m.mcpManager != nil {
		return m.mcpManager.Close()
	}
	return nil
}

// Init app
func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.EnterAltScreen)
}
