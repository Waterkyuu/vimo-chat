package tui

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"vimo-chat/internal/chat"
	"vimo-chat/internal/llm"
	"vimo-chat/internal/memory"
	vimoTools "vimo-chat/internal/tools"
)

// streamChat builds the chat model, ensures the memory service exists, wires the
// tools, opens a streaming chat, and returns a streamStartedMsg carrying the
// live event channel. Any failure is reported via errMsg so the UI can surface
// it without blocking the event loop.
func (m Model) streamChat() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		model, err := llm.NewChatModelProvider(ctx, m.config)
		if err != nil {
			return errMsg{err: err}
		}

		// The memory service (and its SQLite store) is created once and reused.
		memSvc := m.memSvc
		if memSvc == nil {
			memSvc, err = memory.NewMemoryService(model)
			if err != nil {
				return errMsg{err: fmt.Errorf("init memory service: %w", err)}
			}
		}

		tools := vimoTools.LoadTools(ctx, memSvc, m.mcpManager)

		chatSvc := chat.NewChatService(model, tools, memSvc)
		events, err := chatSvc.StreamEvents(ctx, m.messages)
		if err != nil {
			return errMsg{err: fmt.Errorf("start stream: %w", err)}
		}

		return streamStartedMsg{events: events, memSvc: memSvc}
	}
}

// waitForChatEvent reads the next event from the chat channel, blocking until one
// arrives. When the channel is closed (stream finished) it emits streamDoneMsg.
// Each invocation returns a single message; the caller re-arms it to continue
// draining, giving per-event UI updates.
func waitForChatEvent(events <-chan chat.Event) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-events
		if !ok {
			return streamDoneMsg{}
		}
		return chatEventMsg{event: e}
	}
}

// processConversationMemory runs memory extraction and summarization for the
// current conversation in the background so it never blocks the UI. Errors are
// logged and reported via memoryDoneMsg but never interrupt the chat.
func (m Model) processConversationMemory() tea.Cmd {
	memSvc := m.memSvc
	convID := m.convID
	messages := m.messages
	return func() tea.Msg {
		if memSvc == nil || convID == "" {
			return memoryDoneMsg{}
		}
		if err := memSvc.ProcessConversation(context.Background(), convID, messages); err != nil {
			log.Printf("memory: process conversation %s: %v", convID, err)
			return memoryDoneMsg{err: err}
		}
		return memoryDoneMsg{}
	}
}
