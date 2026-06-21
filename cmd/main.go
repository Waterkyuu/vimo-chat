package main

import (
	"fmt"
	"log"
	"os"

	"vimo-chat/internal/skill"
	"vimo-chat/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Load available skills once at startup so they are ready before the first
	// chat request. Skills are additive: a load failure is logged but does not
	// prevent the application from starting.
	if err := skill.LoadAllSkills(); err != nil {
		log.Printf("failed to load skills: %v", err)
	}

	model := tui.NewModel()

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)
	_, err := p.Run()

	// Release long-lived resources (MCP clients) before any exit path.
	if cerr := model.Close(); cerr != nil {
		log.Printf("failed to close mcp: %v", cerr)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
