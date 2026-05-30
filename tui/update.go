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
			if m.mode != modeNormal {
				m.mode = modeNormal
				m.keyInput = ""
				m.textarea.Reset()
				m.refreshLayout()
				return m, nil
			}
			if m.streaming {
				m.streaming = false
				return m, nil
			}
		case tea.KeyUp:
			if m.mode == modeCommandPalette || m.mode == modeProviderMenu || m.mode == modeModelMenu {
				m.moveMenu(-1)
				return m, nil
			}
		case tea.KeyDown:
			if m.mode == modeCommandPalette || m.mode == modeProviderMenu || m.mode == modeModelMenu {
				m.moveMenu(1)
				return m, nil
			}
		case tea.KeyEnter:
			if m.mode == modeCommandPalette {
				m.handleCommand(commandOptions[m.menuIndex].name)
				return m, nil
			}
			if m.mode == modeProviderMenu {
				m.selectProvider()
				return m, nil
			}
			if m.mode == modeModelMenu {
				m.selectModel()
				return m, nil
			}
			if m.mode == modeKeyInput {
				if m.keyInput == "" {
					return m, nil
				}
				m.saveAPIKey()
				return m, nil
			}
			if !m.streaming {
				input := m.textarea.Value()
				if input == "" {
					return m, nil
				}
				if m.handleCommand(input) {
					return m, nil
				}
				if m.activeProviderConfig().APIKey == "" {
					m.textarea.Reset()
					m.setNotice("API key missing. Run /provider, then /key.")
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
		if m.mode == modeKeyInput {
			switch msg.Type {
			case tea.KeyBackspace:
				if len(m.keyInput) > 0 {
					runes := []rune(m.keyInput)
					m.keyInput = string(runes[:len(runes)-1])
				}
			case tea.KeyRunes:
				m.keyInput += string(msg.Runes)
			}
			return m, nil
		}
		if m.mode == modeNormal && msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == '/' &&
			m.textarea.Value() == "" {
			m.mode = modeCommandPalette
			m.menuIndex = 0
			m.textarea, cmd = m.textarea.Update(msg)
			m.refreshLayout()
			return m, cmd
		}

	// Change in window size
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
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
	menuHeight := lineCount(commandPaletteView(*m))

	viewportHeight := height - textareaHeight - statusHeight - separatorHeight - menuHeight

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

func (m *Model) refreshLayout() {
	m.handleResize(m.windowWidth, m.windowHeight)
	m.viewport.GotoBottom()
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	count := 1
	for _, r := range s {
		if r == '\n' {
			count++
		}
	}
	return count
}
