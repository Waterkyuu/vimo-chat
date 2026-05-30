package skill

import (
	"fmt"
	"strings"
)

// Create custom middleware and inject skill descriptions into system prompts.
// This middleware enables skills to be discovered without having to load all their contents in advance.
func BuildSysPrompt() string {
	basePrompt := "You are a professional Markdown and Typst expert, skilled in layout and aesthetic design based on Markdown and Typst.\n" +
		"If the user requests the creation of a paper or resume, use Typst; when the user needs to write notes, use Markdown. \n" +
		"You must wrap the Markdown and Typst content that the user requires using block-level code blocks ```\n" +
		"xxxxxx```" + "\n" + "If you need to invoke the tool, please invoke the tool directly and do not output any analysis text."

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
