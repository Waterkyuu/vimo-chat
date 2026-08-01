package instructions

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestReadFirst_PriorityOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "CLAUDE.md", "claude rules")
	writeFile(t, dir, "AGENTS.md", "agents rules")

	got, ok := readFirst(dir, ruleFiles)
	if !ok {
		t.Fatal("expected a match, got none")
	}
	if got != "agents rules" {
		t.Fatalf("got %q, want %q (AGENTS.md beats CLAUDE.md)", got, "agents rules")
	}
}

func TestReadFirst_SkipsToNextName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "CONTEXT.md", "context rules")

	got, ok := readFirst(dir, ruleFiles)
	if !ok || got != "context rules" {
		t.Fatalf("got (%q,%v), want (%q,true) falling through AGENTS.md/CLAUDE.md", got, ok, "context rules")
	}
}

func TestReadFirst_NoFile(t *testing.T) {
	if _, ok := readFirst(t.TempDir(), ruleFiles); ok {
		t.Fatal("expected no match in empty dir")
	}
}

func TestReadFirst_EmptyDir(t *testing.T) {
	if _, ok := readFirst("", ruleFiles); ok {
		t.Fatal("expected no match for empty dir")
	}
}

func TestFindUp_NearestWins(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a", "b")
	writeFile(t, child, "AGENTS.md", "child rules")
	writeFile(t, root, "AGENTS.md", "root rules")

	got, ok := findUp(child, ruleFiles)
	if !ok || got != "child rules" {
		t.Fatalf("got (%q,%v), want nearest child rules", got, ok)
	}
}

func TestFindUp_FallsBackToParent(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a", "b")
	writeFile(t, root, "AGENTS.md", "root rules")

	got, ok := findUp(child, ruleFiles)
	if !ok || got != "root rules" {
		t.Fatalf("got (%q,%v), want parent root rules", got, ok)
	}
}

// A nearer lower-priority file must beat a farther higher-priority one:
// matching is per-level (nearest first), not global-by-filename.
func TestFindUp_NearestBeatsFartherHigherPriority(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a")
	writeFile(t, child, "CONTEXT.md", "child context")
	writeFile(t, root, "AGENTS.md", "root agents")

	got, ok := findUp(child, ruleFiles)
	if !ok || got != "child context" {
		t.Fatalf("got (%q,%v), want nearest child context", got, ok)
	}
}

func TestFindUp_NothingFound(t *testing.T) {
	root := t.TempDir()
	if _, ok := findUp(filepath.Join(root, "x"), ruleFiles); ok {
		t.Fatal("expected no match when no rule file exists upward")
	}
}

func TestLoadFrom_ProjectOnly(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a")
	writeFile(t, child, "AGENTS.md", "project rules")

	got := LoadFrom(child, t.TempDir())
	if got != "project rules" {
		t.Fatalf("got %q, want project rules only", got)
	}
}

func TestLoadFrom_GlobalOnly(t *testing.T) {
	global := t.TempDir()
	writeFile(t, global, "AGENTS.md", "global rules")

	got := LoadFrom(t.TempDir(), global)
	if got != "global rules" {
		t.Fatalf("got %q, want global rules only", got)
	}
}

func TestLoadFrom_CombinesProjectAndGlobal(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "AGENTS.md", "project rules")
	global := t.TempDir()
	writeFile(t, global, "AGENTS.md", "global rules")

	got := LoadFrom(root, global)
	want := "project rules\n\nglobal rules"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLoadFrom_None(t *testing.T) {
	if got := LoadFrom(t.TempDir(), t.TempDir()); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestLoadFrom_EmptyDirsNoPanic(t *testing.T) {
	if got := LoadFrom("", ""); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

// Load must not panic and must resolve the real environment without error.
// It is a smoke test; exact content depends on the host filesystem.
func TestLoad_DoesNotPanic(t *testing.T) {
	_ = Load()
}
