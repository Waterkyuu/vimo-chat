package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"vimo-chat/internal/chat"
	appconfig "vimo-chat/internal/config"
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

	return Model{
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
}

// Init app
func (m Model) Init() tea.Cmd {
	// Start MCP server
	return tea.Batch(textarea.Blink, tea.EnterAltScreen)
}
