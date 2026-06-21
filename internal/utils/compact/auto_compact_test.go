package compact

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestAutoCompact_ErrModelRequired(t *testing.T) {
	_, err := AutoCompact(
		context.Background(),
		[]*schema.Message{userMsg("a"), assistantMsg("b")},
		nil,
		DefaultOptions(),
	)
	if !errors.Is(err, ErrModelRequired) {
		t.Fatalf("err = %v, want ErrModelRequired", err)
	}
}

func TestAutoCompact_ErrNotEnoughMessages(t *testing.T) {
	_, err := AutoCompact(context.Background(), []*schema.Message{userMsg("a")}, &fakeModel{}, DefaultOptions())
	if !errors.Is(err, ErrNotEnoughMessages) {
		t.Fatalf("err = %v, want ErrNotEnoughMessages", err)
	}
}

func TestAutoCompact_SummarizesAndReplaces(t *testing.T) {
	store := &fakeStore{}
	model := &fakeModel{resp: assistantMsg("This is the summary.")}
	opts := DefaultOptions()
	opts.SummaryStore = store

	// Use a large conversation so the compacted continuation is genuinely
	// smaller than the original history.
	msgs := []*schema.Message{
		userMsg(strings.Repeat("a", 4000)),
		assistantMsg(strings.Repeat("b", 4000)),
		userMsg(strings.Repeat("c", 4000)),
	}

	r, err := AutoCompact(context.Background(), msgs, model, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.Compacted {
		t.Fatal("expected Compacted=true")
	}
	if len(r.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1 (continuation message)", len(r.Messages))
	}
	if r.Messages[0].Role != schema.User {
		t.Fatalf("role = %q, want user", r.Messages[0].Role)
	}
	if !strings.Contains(r.Messages[0].Content, "This is the summary.") {
		t.Fatalf("summary not in continuation message: %q", r.Messages[0].Content)
	}
	if r.TokensAfter >= r.TokensBefore {
		t.Fatalf("expected token reduction: before=%d after=%d", r.TokensBefore, r.TokensAfter)
	}
	if store.saved != "This is the summary." {
		t.Fatalf("summary not persisted: %q", store.saved)
	}
}

func TestAutoCompact_RunsMicroCompactFirst(t *testing.T) {
	opts := Options{
		ProtectedToolResults:      1,
		ProtectedToolResultTokens: 10,
		MinSavingsTokens:          1,
	}
	model := &fakeModel{resp: assistantMsg("summary")}
	msgs := []*schema.Message{
		userMsg("go"),
		assistantToolCall("read", "r1", "{}"),
		toolResult("read", "r1", strings.Repeat("x", 400)), // large old result
		assistantToolCall("read", "r2", "{}"),
		toolResult("read", "r2", "small"),
		userMsg("done"),
	}

	if _, err := AutoCompact(context.Background(), msgs, model, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The model should have received the micro-compacted history, i.e. the old
	// tool result replaced with the placeholder.
	var sawCleared bool
	for _, m := range model.last {
		if m.Content == toolResultClearedPlaceholder {
			sawCleared = true
			break
		}
	}
	if !sawCleared {
		t.Fatal("expected the summarization call to receive micro-compacted history")
	}
}

func TestAutoCompact_EmptySummaryError(t *testing.T) {
	model := &fakeModel{resp: assistantMsg("   ")}
	msgs := []*schema.Message{userMsg("a"), assistantMsg("b")}

	_, err := AutoCompact(context.Background(), msgs, model, DefaultOptions())
	if !errors.Is(err, ErrSummaryEmpty) {
		t.Fatalf("err = %v, want ErrSummaryEmpty", err)
	}
}

func TestAutoCompact_ExtractsSummaryBlock(t *testing.T) {
	resp := assistantMsg("<analysis>thinking...</analysis><summary>real summary</summary>")
	model := &fakeModel{resp: resp}
	store := &fakeStore{}
	opts := DefaultOptions()
	opts.SummaryStore = store

	msgs := []*schema.Message{userMsg("a"), assistantMsg("b")}
	r, err := AutoCompact(context.Background(), msgs, model, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.saved != "real summary" {
		t.Fatalf("saved = %q, want only the summary block", store.saved)
	}
	if strings.Contains(r.Messages[0].Content, "thinking...") {
		t.Fatalf("analysis scratchpad leaked into the continuation message: %q", r.Messages[0].Content)
	}
}

func TestAutoCompact_CustomInstructionsInPrompt(t *testing.T) {
	model := &fakeModel{resp: assistantMsg("summary")}
	opts := DefaultOptions()
	opts.CustomInstructions = "focus on the auth bug fix"

	msgs := []*schema.Message{userMsg("a"), assistantMsg("b")}
	if _, err := AutoCompact(context.Background(), msgs, model, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The last message sent to the model is the structured prompt.
	promptMsg := model.last[len(model.last)-1]
	if !strings.Contains(promptMsg.Content, "focus on the auth bug fix") {
		t.Fatalf("custom instructions missing from prompt: %q", promptMsg.Content)
	}
	if !strings.Contains(promptMsg.Content, "Do NOT use any tools") {
		t.Fatalf("no-tools instruction missing from prompt")
	}
}

func TestAutoCompact_DropsSystemMessages(t *testing.T) {
	model := &fakeModel{resp: assistantMsg("summary")}
	msgs := []*schema.Message{
		{Role: schema.System, Content: "project rules"},
		userMsg("a"),
		assistantMsg("b"),
	}

	if _, err := AutoCompact(context.Background(), msgs, model, DefaultOptions()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// model.last[0] is the dedicated system prompt; the rest must not contain
	// the project rules system message.
	for _, m := range model.last[1:] {
		if m.Role == schema.System {
			t.Fatalf("conversation system message should be dropped from summarization input: %+v", m)
		}
	}
}

func TestAutoCompact_SaveError(t *testing.T) {
	store := &fakeStore{saveErr: errors.New("disk full")}
	model := &fakeModel{resp: assistantMsg("summary")}
	opts := DefaultOptions()
	opts.SummaryStore = store

	msgs := []*schema.Message{userMsg("a"), assistantMsg("b")}
	if _, err := AutoCompact(context.Background(), msgs, model, opts); err == nil {
		t.Fatal("expected error when saving summary fails")
	}
}
