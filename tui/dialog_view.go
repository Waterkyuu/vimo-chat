package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	appconfig "vimo-chat/internal/config"
)

func dialogView(m Model) string {
	switch m.mode {
	case modeProviderMenu:
		return renderPopup("Connect a provider", providerMenuRows(), m.menuIndex+1, m.windowWidth)
	case modeModelMenu:
		return renderPopup("Switch model", modelMenuRows(m.config.ActiveProvider), m.menuIndex+1, m.windowWidth)
	case modeKeyInput:
		return renderKeyPopup(providerLabel(m.keyProvider), m.keyInput, m.windowWidth)
	default:
		return ""
	}
}

func providerMenuRows() []string {
	rows := make([]string, 0, len(providerOptions)+1)
	rows = append(rows, statusStyle.Render("Providers"))
	for _, option := range providerOptions {
		rows = append(rows, option.label)
	}
	return rows
}

func modelMenuRows(provider appconfig.Provider) []string {
	models := currentModelOptions(provider)
	rows := make([]string, 0, len(models)+1)
	rows = append(rows, statusStyle.Render(providerLabel(provider)))
	rows = append(rows, models...)
	return rows
}

func renderPopup(title string, rows []string, selectedRow int, width int) string {
	contentWidth := popupWidth(width) - 4
	var b strings.Builder
	if title != "" {
		b.WriteString(title)
		b.WriteString(strings.Repeat(" ", max(1, contentWidth-lipgloss.Width(title)-3)))
		b.WriteString(statusStyle.Render("esc"))
		b.WriteString("\n\n")
	}
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		if i == selectedRow {
			b.WriteString(selectedStyle.Render(ansi.Truncate(row, contentWidth, "")))
		} else {
			b.WriteString(ansi.Truncate(row, contentWidth, ""))
		}
	}
	return popupStyle(width).Render(b.String())
}

func renderKeyPopup(provider string, keyInput string, width int) string {
	contentWidth := popupWidth(width) - 4
	title := "API key"
	header := title + strings.Repeat(" ", max(1, contentWidth-lipgloss.Width(title)-3)) + statusStyle.Render("esc")
	field := keyInput
	if field == "" {
		field = statusStyle.Render("API key")
	}
	body := fmt.Sprintf("%s\n\n%s\n%s\n\nenter submit", header, provider, ansi.Truncate(field, contentWidth, ""))
	return popupStyle(width).Render(body)
}

func popupWidth(width int) int {
	if width < 48 {
		return max(20, width)
	}
	return min(width-4, 72)
}

func popupStyle(width int) lipgloss.Style {
	return dialogStyle.
		Width(popupWidth(width)).
		Padding(1, 2)
}

func renderDialogLayer(base string, dialog string, width int, height int) string {
	lines := strings.Split(base, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	dialogLines := strings.Split(dialog, "\n")
	start := max(0, (height-len(dialogLines))/2)
	for i, dialogLine := range dialogLines {
		lineIndex := start + i
		if lineIndex >= len(lines) {
			break
		}
		dialogWidth := lipgloss.Width(dialogLine)
		left := max(0, (width-dialogWidth)/2)
		lines[lineIndex] = strings.Repeat(" ", left) + ansi.Truncate(dialogLine, width-left, "")
	}

	return strings.Join(lines, "\n")
}
