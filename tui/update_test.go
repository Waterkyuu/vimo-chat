package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFirstSubmittedUserMessageStaysVisible(t *testing.T) {
	m := NewModel()
	m.showSplash = false

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m = updated.(Model)

	m.textarea.SetValue("first")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	visible := terminalVisibleLines(m.View(), 8)
	if !strings.Contains(visible, "first") {
		t.Fatalf("first submitted message should remain visible; visible output:\n%s", visible)
	}
}

func TestViewFitsWindowHeightAfterResize(t *testing.T) {
	m := NewModel()
	m.showSplash = false

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 8})
	m = updated.(Model)

	if got, want := lineCount(m.View()), 8; got > want {
		t.Fatalf("view rendered %d lines, want at most %d", got, want)
	}
}

func terminalVisibleLines(view string, height int) string {
	lines := strings.Split(view, "\n")
	if len(lines) <= height {
		return view
	}
	return strings.Join(lines[len(lines)-height:], "\n")
}

func lineCount(s string) int {
	return len(strings.Split(s, "\n"))
}
