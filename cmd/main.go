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

	p := tea.NewProgram(
		tui.NewModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
