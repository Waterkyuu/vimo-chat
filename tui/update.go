package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/schema"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Global key processing
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showSplash {
			m.showSplash = false
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			if m.streaming {
				m.streaming = false
				return m, nil
			}
		case tea.KeyEnter:
			if !m.streaming {
				input := m.textarea.Value()
				if input == "" {
					return m, nil
				}
				m.messages = append(m.messages, schema.UserMessage(input))
				m.textarea.Reset()
				m.messages = append(m.messages, schema.AssistantMessage("This is Mock message", nil))
				m.updateViewportContent()
				m.viewport.GotoBottom()
				return m, nil
			}
		}

	// Change in window size
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.handleResize(msg.Width, msg.Height)

		if !m.ready {
			m.ready = true
		}

		return m, nil

	// Stream chunk arrives
	case streamChunkMsg:
		if len(m.messages) > 0 {
			last := m.messages[len(m.messages)-1]
			if last.Role == "assistant" {
				last.Content += msg.chunk
				m.updateViewportContent()
				m.viewport.GotoBottom()
			}
		}
		return m, nil

	// Stream completion
	case streamDoneMsg:
		m.streaming = false
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)

}

func (m *Model) handleResize(width, height int) {
	textareaHeight := 2
	statusHeight := 1
	separatorHeight := 2

	viewportHeight := height - textareaHeight - statusHeight - separatorHeight

	if viewportHeight < 1 {
		viewportHeight = 1
	}

	m.viewport = viewport.New(width, viewportHeight)
	m.viewport.SetContent(renderMessage(m.messages))
	m.textarea.SetWidth(width)
	m.textarea.SetHeight(textareaHeight)
}

func (m *Model) updateViewportContent() {
	m.viewport.SetContent(renderMessage(m.messages))
}
