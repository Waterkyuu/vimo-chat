package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/schema"
)

var (
	userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)

	assistantStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func (m Model) View() string {
	if m.showSplash {
		return splashScreen(m.windowWidth)
	}

	if !m.ready {
		return "Initializing"
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		separator(m.windowWidth),
		m.textarea.View(),
		separator(m.windowWidth),
		statusBar(m),
	)
}

// Convert the message list into viewport content for update invocation
func renderMessage(messages []*schema.Message) string {
	var b strings.Builder
	for i, msg := range messages {
		switch msg.Role {
		case "user":
			b.WriteString(userStyle.Render("You: "))
			b.WriteString(msg.Content)

		case "assistant":
			b.WriteString(assistantStyle.Render("Vimo: "))
			b.WriteString(msg.Content)
		}

		if i < len(messages)-1 {
			b.WriteString("\n\n")
		}
	}

	return b.String()
}

func separator(width int) string {
	return strings.Repeat("─", width)
}

func statusBar(m Model) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Message: %d", len(m.messages)))

	if m.streaming {
		parts = append(parts, "● Streaming...")
	}

	if m.err != nil {
		parts = append(parts, errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return statusStyle.Render(strings.Join(parts, " | "))
}

func splashScreen(width int) string {
	logo := `
██╗   ██╗███╗███╗   ███╗██████╗
██║   ██║██╔╝████╗ ████║██╔══██╗
██║   ██║██║ ██╔████╔██║██║  ██║
╚██╗ ██╔╝██║ ██║╚██╔╝██║██║  ██║
 ╚████╔╝ ██║ ██║ ╚═╝ ██║╚█████╔╝
  ╚═══╝  ╚═╝ ╚═╝     ╚═╝ ╚════╝`

	subtitle := "AI Chat with Memory · v0.1.0"

	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1E3A5F")).
		Bold(true)

	subStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6"))

	content := lipgloss.NewStyle().
		Width(width).
		Height(20).
		Align(lipgloss.Center, lipgloss.Center).
		Render(
			logoStyle.Render(logo) + "\n\n" +
				subStyle.Render(subtitle) + "\n\n" +
				hintStyle.Render("Press any key to start..."),
		)

	return content
}
