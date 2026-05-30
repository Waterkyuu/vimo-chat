package skill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	Name        string
	Description string
	Content     string
}

var Skills []Skill

// loadAllSkills reads all SKILL.md files from the skills directory when server start
func LoadAllSkills() error {
	// Get the project root directory
	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	skillsDir := filepath.Join(projectRoot, "skills")

	// Walk through the skills directory
	err = filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process SKILL.md files
		if filepath.Base(path) == "SKILL.md" {
			skill, err := parseSkillFile(path)
			if err != nil {
				return fmt.Errorf("failed to parse skill file %s: %w", path, err)
			}
			Skills = append(Skills, skill)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk skills directory: %w", err)
	}

	return nil
}

// parseSkillFile reads and parses a single SKILL.md file
func parseSkillFile(filePath string) (Skill, error) {
	// Validate file path to prevent path traversal attacks
	// Ensure the file path is within the skills directory
	projectRoot, err := getProjectRoot()
	if err != nil {
		return Skill{}, fmt.Errorf("failed to get project root: %w", err)
	}

	skillsDir := filepath.Join(projectRoot, "skills")
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return Skill{}, fmt.Errorf("failed to get absolute path: %w", err)
	}

	absSkillsDir, err := filepath.Abs(skillsDir)
	if err != nil {
		return Skill{}, fmt.Errorf("failed to get absolute skills directory: %w", err)
	}

	// Check if the file path is within the skills directory
	relPath, err := filepath.Rel(absSkillsDir, absFilePath)
	if err != nil {
		return Skill{}, fmt.Errorf("invalid file path: %w", err)
	}

	// Prevent path traversal: the relative path should not start with ".."
	if strings.HasPrefix(relPath, "..") {
		return Skill{}, fmt.Errorf("potential path traversal attack detected: %s", filePath)
	}

	// Read the validated absolute path
	// #nosec G304 - Path is validated above to prevent path traversal attacks
	content, err := os.ReadFile(absFilePath)
	if err != nil {
		return Skill{}, fmt.Errorf("failed to read file: %w", err)
	}

	// Extract skill name from directory path
	skillName := filepath.Base(filepath.Dir(filePath))

	// Parse frontmatter and content
	name, description, contentWithoutFrontmatter := parseFrontmatter(string(content))

	// Use frontmatter name if available, otherwise use directory name
	if name != "" {
		skillName = name
	}

	return Skill{
		Name:        skillName,
		Description: description,
		Content:     contentWithoutFrontmatter,
	}, nil
}

// parseFrontmatter parses YAML frontmatter from markdown content without external dependencies
func parseFrontmatter(content string) (name, description, contentWithoutFrontmatter string) {
	lines := strings.Split(content, "\n")

	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		// No frontmatter, return original content
		return "", "", content
	}

	// Extract frontmatter lines
	var frontmatterLines []string
	frontmatterEnd := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			// Found end of frontmatter
			frontmatterEnd = i
			// Collect content after frontmatter
			if i+1 < len(lines) {
				contentWithoutFrontmatter = strings.Join(lines[i+1:], "\n")
			}
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	if len(frontmatterLines) == 0 || frontmatterEnd == -1 {
		return "", "", content
	}

	// Parse simple YAML frontmatter (only name and description)
	for _, line := range frontmatterLines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			// Remove quotes if present
			name = strings.Trim(name, "\"'`")
		} else if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			// Remove quotes if present
			description = strings.Trim(description, "\"'`")
		}
	}

	return name, description, contentWithoutFrontmatter
}

// getProjectRoot returns the project root directory
func getProjectRoot() (string, error) {
	// Start from current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up to find go.mod file (project root)
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			return "", fmt.Errorf("could not find project root (go.mod not found)")
		}
		dir = parent
	}
}
