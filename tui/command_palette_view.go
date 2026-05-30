package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func commandPaletteView(m Model) string {
	if m.mode != modeCommandPalette {
		return ""
	}

	rows := make([]string, 0, len(commandOptions))
	contentWidth := max(20, m.windowWidth-2)
	for i, option := range commandOptions {
		row := " " + ansi.Truncate(fmt.Sprintf("%-12s %s", option.name, option.description), contentWidth-1, "")
		if i == m.menuIndex {
			row = selectedStyle.Render(row)
		}
		rows = append(rows, row)
	}
	rows = append(rows, strings.Repeat("─", m.windowWidth))

	return strings.Join(rows, "\n")
}
