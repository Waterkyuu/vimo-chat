package skill

import (
	"fmt"
	"strings"
)

// Create custom middleware and inject skill descriptions into system prompts.
// This middleware enables skills to be discovered without having to load all their contents in advance.
func BuildSysPrompt() string {
	basePrompt := "You are a helpful assistant"

	var skillList []string
	for _, skill := range Skills {
		skillList = append(skillList, fmt.Sprintf("- **%s**: %s", skill.Name, skill.Description))
	}
	skillsPrompt := strings.Join(skillList, "\n")

	// Piece together the final System Prompt
	fullPrompt := fmt.Sprintf(`%s
		## Available Skills
		%s
		Use the load_skill tool when you need detailed information about handling a specific type of request.`, basePrompt, skillsPrompt)

	return fullPrompt
}
