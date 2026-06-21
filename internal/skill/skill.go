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

// LoadAllSkills reads every SKILL.md file from the configured skill directories
// when the server starts. Built-in skills ship with the project, while
// project-local skills are read from ".agents/skills" in the user's current
// working directory, mirroring the convention used by mainstream coding agents.
func LoadAllSkills() error {
	// Reset the registry so repeated calls stay idempotent.
	loaded, err := loadSkills(skillDirs())
	if err != nil {
		return err
	}
	Skills = loaded
	return nil
}

// skillDirs returns the skill directories to scan, ordered from lowest to
// highest priority. A directory resolved later overrides an earlier one when
// two skills share the same name. Directories that cannot be resolved are
// skipped silently.
func skillDirs() []string {
	var dirs []string

	// Built-in skills bundled with the project.
	if projectRoot, err := getProjectRoot(); err == nil {
		dirs = append(dirs, filepath.Join(projectRoot, "skills"))
	}

	// Project-local skills in the user's current working directory.
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, ".agents", "skills"))
	}

	return dirs
}

// loadSkills walks the given directories in priority order and parses every
// SKILL.md file. Missing directories are skipped. On a name collision the skill
// from a later (higher priority) directory replaces the earlier one.
func loadSkills(dirs []string) ([]Skill, error) {
	var skills []Skill
	indexByName := make(map[string]int)

	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			// Directory does not exist or is inaccessible; skip silently.
			continue
		}

		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Base(path) != "SKILL.md" {
				return nil
			}

			skill, err := parseSkillFile(path, dir)
			if err != nil {
				return fmt.Errorf("failed to parse skill file %s: %w", path, err)
			}

			key := strings.ToLower(skill.Name)
			if idx, ok := indexByName[key]; ok {
				// Higher priority directory overrides an existing skill.
				skills[idx] = skill
			} else {
				indexByName[key] = len(skills)
				skills = append(skills, skill)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk skills directory %s: %w", dir, err)
		}
	}

	return skills, nil
}

// parseSkillFile reads and parses a single SKILL.md file. baseDir is the owning
// skills directory and is used to validate that filePath stays within it, which
// prevents path traversal attacks.
func parseSkillFile(filePath, baseDir string) (Skill, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return Skill{}, fmt.Errorf("failed to get absolute path: %w", err)
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return Skill{}, fmt.Errorf("failed to get absolute skills directory: %w", err)
	}

	// Check if the file path is within the skills directory.
	relPath, err := filepath.Rel(absBaseDir, absFilePath)
	if err != nil {
		return Skill{}, fmt.Errorf("invalid file path: %w", err)
	}

	// Prevent path traversal: the relative path should not start with "..".
	if strings.HasPrefix(relPath, "..") {
		return Skill{}, fmt.Errorf("potential path traversal attack detected: %s", filePath)
	}

	// Read the validated absolute path.
	// #nosec G304 - Path is validated above to prevent path traversal attacks.
	content, err := os.ReadFile(absFilePath)
	if err != nil {
		return Skill{}, fmt.Errorf("failed to read file: %w", err)
	}

	// Extract skill name from directory path.
	skillName := filepath.Base(filepath.Dir(filePath))

	// Parse frontmatter and content.
	name, description, contentWithoutFrontmatter := parseFrontmatter(string(content))

	// Use frontmatter name if available, otherwise use directory name.
	if name != "" {
		skillName = name
	}

	return Skill{
		Name:        skillName,
		Description: description,
		Content:     contentWithoutFrontmatter,
	}, nil
}

// parseFrontmatter parses YAML frontmatter from markdown content without external dependencies.
func parseFrontmatter(content string) (name, description, contentWithoutFrontmatter string) {
	lines := strings.Split(content, "\n")

	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		// No frontmatter, return original content.
		return "", "", content
	}

	// Extract frontmatter lines.
	var frontmatterLines []string
	frontmatterEnd := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			// Found end of frontmatter.
			frontmatterEnd = i
			// Collect content after frontmatter.
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

	// Parse simple YAML frontmatter (only name and description).
	for _, line := range frontmatterLines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			// Remove quotes if present.
			name = strings.Trim(name, "\"'`")
		} else if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			// Remove quotes if present.
			description = strings.Trim(description, "\"'`")
		}
	}

	return name, description, contentWithoutFrontmatter
}

// getProjectRoot returns the project root directory.
func getProjectRoot() (string, error) {
	// Start from current working directory.
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up to find go.mod file (project root).
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod.
			return "", fmt.Errorf("could not find project root (go.mod not found)")
		}
		dir = parent
	}
}
