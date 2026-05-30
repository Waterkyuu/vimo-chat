package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/schema"
)

// Message type
type streamChunkMsg struct {
	chunk string
}

type streamDoneMsg struct{}

type errMsg struct {
	err error
}

// Data model
type Model struct {
	// UI comp
	viewport viewport.Model
	textarea textarea.Model
	// Data
	messages []*schema.Message
	err      error
	// Status mark
	streaming   bool
	ready       bool
	showSplash  bool
	windowWidth int
}

func NewModel() Model {
	ta := textarea.New()
	ta.Placeholder = "Send a message... (Enter to send)"
	ta.ShowLineNumbers = false
	ta.Focus()

	ta.Prompt = "| "
	ta.CharLimit = 10000

	return Model{
		textarea:    ta,
		messages:    []*schema.Message{},
		showSplash:  true,
		windowWidth: 80,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.EnterAltScreen)
}
