package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"vimo-chat/internal/chat"
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
			// if m.streaming {
			// 	m.streaming = false
			// 	return m, nil
			// }
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
				m.streaming = true
				m.err = nil
				m.assistantText = ""
				if m.convID == "" {
					m.convID = uuid.NewString()
				}
				m.updateViewportContent()
				m.viewport.GotoBottom()
				return m, m.streamChat()
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

	case toolLoadMsg:
		m.tools = msg.tools
		if msg.err != nil {
			m.setNotice(fmt.Sprintf("MCP failed: %v", msg.err))
		}

	// The chat stream has been opened: cache the memory service and the event
	// channel, then start draining events one at a time for live updates.
	case streamStartedMsg:
		m.memSvc = msg.memSvc
		m.streamEvents = msg.events
		return m, waitForChatEvent(msg.events)

	// Stream chunk arrives.
	case chatEventMsg:
		switch msg.event.Type {
		case chat.EventAssistantDelta:
			m.assistantText += msg.event.Content
			m.updateViewportContent()
			m.viewport.GotoBottom()
			return m, waitForChatEvent(m.streamEvents)
		case chat.EventToolStart, chat.EventToolResult:
			return m, waitForChatEvent(m.streamEvents)
		case chat.EventDone:
			// Finalization happens on streamDoneMsg once the channel is closed.
			return m, waitForChatEvent(m.streamEvents)
		case chat.EventError:
			m.streaming = false
			m.err = msg.event.Err
			return m, nil
		}

	// Stream completion: persist the assembled assistant reply and trigger
	// background memory extraction/summarization for this conversation.
	case streamDoneMsg:
		if m.assistantText != "" {
			m.messages = append(m.messages, schema.AssistantMessage(m.assistantText, nil))
			m.assistantText = ""
		}
		m.streaming = false
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return m, m.processConversationMemory()

	// Background memory processing finished; errors are already logged and
	// non-fatal, so just absorb the message.
	case memoryDoneMsg:
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
	m.updateViewportContent()
	m.textarea.SetWidth(width)
	m.textarea.SetHeight(textareaHeight)
}

// updateViewportContent renders the committed messages plus any in-progress
// assistant reply (while streaming) so partial responses are visible live.
func (m *Model) updateViewportContent() {
	msgs := m.messages
	if m.streaming && m.assistantText != "" {
		msgs = append(append([]*schema.Message{}, m.messages...),
			schema.AssistantMessage(m.assistantText, nil))
	}
	m.viewport.SetContent(renderMessage(msgs))
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
