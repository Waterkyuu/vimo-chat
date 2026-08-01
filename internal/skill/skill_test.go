package skill

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSkill writes a SKILL.md file into dir/<name>/SKILL.md.
func writeSkill(t *testing.T, dir, name, body string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", skillDir, err)
	}
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantName string
		wantDesc string
		wantBody string
	}{
		{
			name:     "single line fields",
			content:  "---\nname: my-skill\ndescription: does a thing\n---\n\n# Body\n",
			wantName: "my-skill",
			wantDesc: "does a thing",
			wantBody: "\n# Body\n",
		},
		{
			name:     "quoted values",
			content:  "---\nname: \"quoted\"\ndescription: 'desc'\n---\nbody",
			wantName: "quoted",
			wantDesc: "desc",
			wantBody: "body",
		},
		{
			name:     "no frontmatter",
			content:  "# Just a title\n\ntext",
			wantName: "",
			wantDesc: "",
			wantBody: "# Just a title\n\ntext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotDesc, gotBody := parseFrontmatter(tt.content)
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotDesc != tt.wantDesc {
				t.Errorf("description = %q, want %q", gotDesc, tt.wantDesc)
			}
			if gotBody != tt.wantBody {
				t.Errorf("body = %q, want %q", gotBody, tt.wantBody)
			}
		})
	}
}

func TestParseSkillFile_FallsBackToDirName(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "from-dir", "# No frontmatter here\n")

	got, err := parseSkillFile(filepath.Join(dir, "from-dir", "SKILL.md"), dir)
	if err != nil {
		t.Fatalf("parseSkillFile: %v", err)
	}
	if got.Name != "from-dir" {
		t.Errorf("Name = %q, want from-dir", got.Name)
	}
}

func TestParseSkillFile_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "SKILL.md")
	if err := os.WriteFile(target, []byte("---\nname: evil\n---\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := parseSkillFile(target, dir); err == nil {
		t.Fatal("expected path traversal error, got nil")
	}
}

func TestLoadSkills_ReadsAllDirectories(t *testing.T) {
	builtin := t.TempDir()
	agents := t.TempDir()

	writeSkill(t, builtin, "built-in-only", "---\nname: built-in-only\ndescription: a\n---\nbody A")
	writeSkill(t, builtin, "shared", "---\nname: shared\ndescription: from-built-in\n---\nbody shared built-in")
	writeSkill(t, agents, "agents-only", "---\nname: agents-only\ndescription: b\n---\nbody B")
	writeSkill(t, agents, "shared", "---\nname: shared\ndescription: from-agents\n---\nbody shared agents")

	skills, err := loadSkills([]string{builtin, agents})
	if err != nil {
		t.Fatalf("loadSkills: %v", err)
	}

	byName := make(map[string]Skill, len(skills))
	for _, s := range skills {
		byName[s.Name] = s
	}

	if len(byName) != 3 {
		t.Fatalf("got %d unique skills, want 3 (%v)", len(byName), byName)
	}
	if _, ok := byName["built-in-only"]; !ok {
		t.Error("missing built-in-only")
	}
	if _, ok := byName["agents-only"]; !ok {
		t.Error("missing agents-only")
	}
	// Later directory (agents) overrides the earlier (built-in) on collision.
	if byName["shared"].Description != "from-agents" {
		t.Errorf("shared overridden incorrectly: %q", byName["shared"].Description)
	}
}

func TestLoadSkills_SkipsMissingDirectories(t *testing.T) {
	present := t.TempDir()
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	writeSkill(t, present, "solo", "---\nname: solo\ndescription: only\n---\nbody")

	skills, err := loadSkills([]string{missing, present})
	if err != nil {
		t.Fatalf("loadSkills: %v", err)
	}
	if len(skills) != 1 || skills[0].Name != "solo" {
		t.Fatalf("got %+v, want one skill named solo", skills)
	}
}
