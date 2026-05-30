package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/schema"

	appconfig "vimo-chat/internal/config"
)

// Message type
type streamChunkMsg struct {
	chunk string
}

type streamDoneMsg struct{}

type errMsg struct {
	err error
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

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.EnterAltScreen)
}
