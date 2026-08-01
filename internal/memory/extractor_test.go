package memory

import (
	"testing"
)

func TestParseExtractedMemories_CleanJSON(t *testing.T) {
	raw := `[{"content":"User prefers dark theme"},{"content":"User writes Go"}]`

	got, err := parseExtractedMemories(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	want := []string{"User prefers dark theme", "User writes Go"}
	for i, m := range got {
		if m.Content != want[i] {
			t.Errorf("mem[%d].Content = %q, want %q", i, m.Content, want[i])
		}
		if m.Type != MemoryTypeExtractedKnowledge {
			t.Errorf("mem[%d].Type = %q, want %q", i, m.Type, MemoryTypeExtractedKnowledge)
		}
		if m.ID == "" {
			t.Errorf("mem[%d].ID is empty", i)
		}
	}
}

func TestParseExtractedMemories_MarkdownFence(t *testing.T) {
	raw := "```json\n[\n  {\"content\": \"likes Vim\"}\n]\n```"

	got, err := parseExtractedMemories(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Content != "likes Vim" {
		t.Fatalf("got = %+v, want one memory with content \"likes Vim\"", got)
	}
}

func TestParseExtractedMemories_ProseAround(t *testing.T) {
	raw := "Here are the memories:\n[{\"content\":\"prefers tabs\"}]\nThat's all."

	got, err := parseExtractedMemories(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Content != "prefers tabs" {
		t.Fatalf("got = %+v, want one memory with content \"prefers tabs\"", got)
	}
}

func TestParseExtractedMemories_EmptyArray(t *testing.T) {
	for _, raw := range []string{"[]", "  []  ", "```json\n[]\n```"} {
		got, err := parseExtractedMemories(raw)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", raw, err)
		}
		if got != nil {
			t.Fatalf("for %q got %v, want nil", raw, got)
		}
	}
}

func TestParseExtractedMemories_SkipsBlankContent(t *testing.T) {
	raw := `[{"content":"valid"},{"content":"   "}]`

	got, err := parseExtractedMemories(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (blank content skipped)", len(got))
	}
	if got[0].Content != "valid" {
		t.Fatalf("content = %q, want %q", got[0].Content, "valid")
	}
}

func TestParseExtractedMemories_InvalidJSON(t *testing.T) {
	raw := "this is not json at all"

	if _, err := parseExtractedMemories(raw); err == nil {
		t.Fatal("want error for invalid JSON, got nil")
	}
}

func TestParseExtractedMemories_UniqueIDs(t *testing.T) {
	raw := `[{"content":"a"},{"content":"b"}]`

	got, err := parseExtractedMemories(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].ID == got[1].ID {
		t.Fatalf("duplicate IDs: %s", got[0].ID)
	}
}
