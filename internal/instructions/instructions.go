// Package instructions resolves and combines project and global rule files
// (AGENTS.md and its aliases) into a single text block for the system prompt.
//
// The resolution follows the OpenCode convention:
//
//   - Project rules: traverse upward from the current working directory and use
//     the first matching file among AGENTS.md, CLAUDE.md, CONTEXT.md (highest
//     priority first).
//   - Global rules: <os.UserConfigDir()>/vimo/AGENTS.md.
//   - Project and global rules are combined (project first); missing files are
//     silently skipped.
package instructions

import (
	"os"
	"path/filepath"
	"strings"
)

// ruleFiles is the priority-ordered list of recognized project rule filenames.
var ruleFiles = []string{"AGENTS.md", "CLAUDE.md", "CONTEXT.md"}

// globalRuleFiles is the list of recognized global rule filenames. Only the
// vimo-specific location is checked to keep global resolution predictable.
var globalRuleFiles = []string{"AGENTS.md"}

// Load resolves instructions from vimo's default locations: the current working
// directory (traversed upward) for project rules and os.UserConfigDir/vimo for
// global rules. It returns the combined text, or "" when no rule file exists.
func Load() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return LoadFrom(cwd, "")
	}
	return LoadFrom(cwd, filepath.Join(base, "vimo"))
}

// LoadFrom resolves instructions rooted at cwd for project rules and globalDir
// for global rules. It is split out from Load so the filesystem-independent
// resolution logic can be unit tested without touching os.Getwd or
// os.UserConfigDir.
//
// Project and global results are concatenated (project first) separated by a
// blank line. An empty result means no rule file was found anywhere.
func LoadFrom(cwd, globalDir string) string {
	var parts []string

	if content, ok := findUp(cwd, ruleFiles); ok {
		parts = append(parts, content)
	}
	if content, ok := readFirst(globalDir, globalRuleFiles); ok {
		parts = append(parts, content)
	}

	return strings.Join(parts, "\n\n")
}

// findUp traverses upward from start, checking each directory for the first
// existing file among names (in priority order). It returns the trimmed
// content of the first match. Traversal stops at the filesystem root.
func findUp(start string, names []string) (string, bool) {
	dir := start
	for dir != "" {
		if content, ok := readFirst(dir, names); ok {
			return content, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// readFirst returns the trimmed content of the first existing file among names
// in dir, checked in the given order. A non-existent or unreadable file is
// skipped so a higher-priority miss falls through to the next candidate.
func readFirst(dir string, names []string) (string, bool) {
	if dir == "" {
		return "", false
	}
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			return strings.TrimSpace(string(data)), true
		}
	}
	return "", false
}
