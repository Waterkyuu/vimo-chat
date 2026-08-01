package compact

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Compile-time assertion that the real OpenAI chat model satisfies the
// compact.Model interface (variadic model.Option and all).
var _ Model = (*openai.ChatModel)(nil)

// --- Test doubles shared across the package tests ---

type fakeModel struct {
	resp *schema.Message
	err  error
	last []*schema.Message
}

func (f *fakeModel) Generate(
	_ context.Context,
	messages []*schema.Message,
	_ ...model.Option,
) (*schema.Message, error) {
	f.last = messages
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

type fakeStore struct {
	summary string
	ok      bool
	err     error

	saved   string
	saveErr error
}

func (f *fakeStore) GetSessionSummary(_ context.Context) (string, bool, error) {
	return f.summary, f.ok, f.err
}

func (f *fakeStore) SaveSessionSummary(_ context.Context, summary string) error {
	f.saved = summary
	return f.saveErr
}

// --- Message builders ---

func userMsg(content string) *schema.Message {
	return &schema.Message{Role: schema.User, Content: content}
}

func assistantMsg(content string) *schema.Message {
	return &schema.Message{Role: schema.Assistant, Content: content}
}

func assistantToolCall(name, id, args string) *schema.Message {
	return &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID:       id,
			Function: schema.FunctionCall{Name: name, Arguments: args},
		}},
	}
}

func toolResult(name, id, content string) *schema.Message {
	return &schema.Message{
		Role:       schema.Tool,
		ToolName:   name,
		ToolCallID: id,
		Content:    content,
	}
}

// --- Token estimation ---

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Fatalf("empty = %d, want 0", got)
	}
	// 8 chars -> 2 tokens.
	if got := EstimateTokens("abcdefgh"); got != 2 {
		t.Fatalf("8 chars = %d, want 2", got)
	}
	// Rounding up: 1 char -> 1 token.
	if got := EstimateTokens("a"); got != 1 {
		t.Fatalf("1 char = %d, want 1", got)
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	m := &schema.Message{
		Role:    schema.Assistant,
		Content: "hello world!", // 12 chars -> 3 tokens
		ToolCalls: []schema.ToolCall{{
			Function: schema.FunctionCall{Name: "read", Arguments: `{"path":"a.go"}`}, // 4 + 13 chars -> ~5 tokens
		}},
	}
	got := EstimateMessageTokens(m)
	if got <= 0 {
		t.Fatalf("expected positive token estimate, got %d", got)
	}
}

func TestOptionsThresholds(t *testing.T) {
	opts := Options{ContextWindow: 200_000, ReservedOutputTokens: 8_000}
	// Effective window reserves output tokens.
	if got := opts.EffectiveWindow(); got != 192_000 {
		t.Fatalf("EffectiveWindow = %d, want 192000", got)
	}
	// Auto-compact fires 13K below the effective window.
	if got := opts.AutoCompactThreshold(); got != 179_000 {
		t.Fatalf("AutoCompactThreshold = %d, want 179000", got)
	}
	// Warning shows 20K below the auto-compact trigger.
	if got := opts.WarningThreshold(); got != 159_000 {
		t.Fatalf("WarningThreshold = %d, want 159000", got)
	}
	// Hard stop is 3K below the full window.
	if got := opts.BlockingLimit(); got != 197_000 {
		t.Fatalf("BlockingLimit = %d, want 197000", got)
	}
}

func TestOptionsNormalizedFillsDefaults(t *testing.T) {
	o := Options{}.normalized()
	if o.ContextWindow != DefaultContextWindow {
		t.Fatalf("ContextWindow = %d, want default", o.ContextWindow)
	}
	if o.CompactableTools == nil {
		t.Fatal("CompactableTools should default to a non-nil set")
	}
	if !isCompactableTool("Read", o.CompactableTools) { // case-insensitive
		t.Fatal("Read should be compactable by default")
	}
}

func TestShouldAutoCompact(t *testing.T) {
	opts := Options{ContextWindow: 200_000, ReservedOutputTokens: 20_000} // threshold = 167000

	// Small session: no compaction.
	if ShouldAutoCompact([]*schema.Message{userMsg("hi")}, opts) {
		t.Fatal("small session should not trigger auto-compact")
	}

	// Huge session: compaction.
	big := strings.Repeat("x", 200_000*4) // ~200k tokens
	if !ShouldAutoCompact([]*schema.Message{userMsg(big)}, opts) {
		t.Fatal("large session should trigger auto-compact")
	}
}

func TestBuildContinuationMessage(t *testing.T) {
	m := buildContinuationMessage("the summary")
	if m.Role != schema.User {
		t.Fatalf("role = %q, want user", m.Role)
	}
	if !strings.Contains(m.Content, "Conversation compacted") {
		t.Fatalf("missing boundary marker: %q", m.Content)
	}
	if !strings.Contains(m.Content, "the summary") {
		t.Fatalf("missing summary body: %q", m.Content)
	}
	if !strings.Contains(m.Content, "continue the conversation") {
		t.Fatalf("missing continuation instruction: %q", m.Content)
	}
}

func TestBetween(t *testing.T) {
	if v, ok := between("a<summary>x</summary>b", "<summary>", "</summary>"); !ok || v != "x" {
		t.Fatalf("got (%q,%v), want (x,true)", v, ok)
	}
	if _, ok := between("no tags here", "<summary>", "</summary>"); ok {
		t.Fatal("expected ok=false when tags missing")
	}
	if _, ok := between("only <summary> open", "<summary>", "</summary>"); ok {
		t.Fatal("expected ok=false when close tag missing")
	}
}

func TestFormatConversationToolCalls(t *testing.T) {
	msgs := []*schema.Message{
		assistantToolCall("read", "r1", `{"path":"a.go"}`),
	}
	got := formatConversation(msgs)
	if !strings.Contains(got, "tool_call read") {
		t.Fatalf("expected tool_call rendered, got %q", got)
	}
}
