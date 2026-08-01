package compact

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestMicroCompact_NoToolResults(t *testing.T) {
	msgs := []*schema.Message{userMsg("hi"), assistantMsg("hello")}
	r := MicroCompact(msgs, DefaultOptions())
	if r.Compacted {
		t.Fatal("expected Compacted=false with no tool results")
	}
	if r.TokensSaved != 0 {
		t.Fatalf("TokensSaved = %d, want 0", r.TokensSaved)
	}
}

func TestMicroCompact_HotTailProtected(t *testing.T) {
	// Three tool results: all protected by the default hot tail (3), so none
	// are cleared even though they are large.
	opts := Options{
		ProtectedToolResults:      3,
		ProtectedToolResultTokens: 1,
		MinSavingsTokens:          1,
	}
	msgs := []*schema.Message{
		userMsg("go"),
		assistantToolCall("read", "r1", "{}"),
		toolResult("read", "r1", strings.Repeat("a", 400)),
		assistantToolCall("read", "r2", "{}"),
		toolResult("read", "r2", strings.Repeat("b", 400)),
		assistantToolCall("read", "r3", "{}"),
		toolResult("read", "r3", strings.Repeat("c", 400)),
	}

	r := MicroCompact(msgs, opts)
	if r.Compacted {
		t.Fatal("expected all tool results protected (hot tail)")
	}
	for _, m := range r.Messages {
		if m.Content == toolResultClearedPlaceholder {
			t.Fatal("no content should be cleared when all results are in the hot tail")
		}
	}
}

func TestMicroCompact_BelowMinSavings(t *testing.T) {
	// Eligible tokens exist but are smaller than MinSavingsTokens.
	opts := Options{
		ProtectedToolResults:      1,
		ProtectedToolResultTokens: 1,
		MinSavingsTokens:          1_000,
	}
	msgs := []*schema.Message{
		userMsg("go"),
		assistantToolCall("read", "r1", "{}"),
		toolResult("read", "r1", "small result"),
		assistantToolCall("read", "r2", "{}"),
		toolResult("read", "r2", "small result"),
	}

	r := MicroCompact(msgs, opts)
	if r.Compacted {
		t.Fatal("expected no compaction when savings below threshold")
	}
}

func TestMicroCompact_ClearsOldResults(t *testing.T) {
	opts := Options{
		ProtectedToolResults:      1,
		ProtectedToolResultTokens: 10,
		MinSavingsTokens:          1,
	}
	msgs := []*schema.Message{
		userMsg("go"),
		assistantToolCall("read", "r1", "{}"),
		toolResult("read", "r1", strings.Repeat("x", 400)), // old + large -> eligible
		assistantToolCall("read", "r2", "{}"),
		toolResult("read", "r2", "recent small"), // protected hot tail
		userMsg("done"),
	}

	r := MicroCompact(msgs, opts)
	if !r.Compacted {
		t.Fatal("expected Compacted=true")
	}
	if r.TokensSaved <= 0 {
		t.Fatalf("TokensSaved = %d, want > 0", r.TokensSaved)
	}

	cleared := r.Messages[2]
	if cleared.Content != toolResultClearedPlaceholder {
		t.Fatalf("old tool result not cleared: %q", cleared.Content)
	}
	if cleared.Role != schema.Tool {
		t.Fatalf("cleared message role = %q, want tool", cleared.Role)
	}

	kept := r.Messages[4]
	if kept.Content != "recent small" {
		t.Fatalf("hot-tail result should be kept intact: %q", kept.Content)
	}
}

func TestMicroCompact_NonCompactableToolUntouched(t *testing.T) {
	opts := Options{
		ProtectedToolResults:      1,
		ProtectedToolResultTokens: 10,
		MinSavingsTokens:          1,
		CompactableTools:          map[string]struct{}{"read": {}},
	}
	msgs := []*schema.Message{
		userMsg("go"),
		assistantToolCall("memory_save", "m1", "{}"),
		toolResult("memory_save", "m1", strings.Repeat("y", 400)), // not compactable
		userMsg("done"),
	}

	r := MicroCompact(msgs, opts)
	if r.Compacted {
		t.Fatal("non-compactable tool results must never be cleared")
	}
}

func TestMicroCompact_DoesNotMutateInput(t *testing.T) {
	opts := Options{
		ProtectedToolResults:      1,
		ProtectedToolResultTokens: 10,
		MinSavingsTokens:          1,
	}
	original := strings.Repeat("z", 400)
	msgs := []*schema.Message{
		userMsg("go"),
		assistantToolCall("read", "r1", "{}"),
		toolResult("read", "r1", original),
		assistantToolCall("read", "r2", "{}"),
		toolResult("read", "r2", "small"),
	}

	_ = MicroCompact(msgs, opts)
	if msgs[2].Content != original {
		t.Fatalf("input was mutated: %q", msgs[2].Content)
	}
}

func TestMicroCompact_CaseInsensitiveToolName(t *testing.T) {
	opts := Options{
		ProtectedToolResults:      1,
		ProtectedToolResultTokens: 10,
		MinSavingsTokens:          1,
	}
	msgs := []*schema.Message{
		userMsg("go"),
		assistantToolCall("READ", "r1", "{}"), // uppercase tool name
		toolResult("Read", "r1", strings.Repeat("x", 400)),
		assistantToolCall("read", "r2", "{}"),
		toolResult("read", "r2", "small"),
	}

	r := MicroCompact(msgs, opts)
	if !r.Compacted {
		t.Fatal("expected case-insensitive match to clear the old READ result")
	}
}
