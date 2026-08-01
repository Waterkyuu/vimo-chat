// Package compact implements Claude Code style context compaction.
//
// It provides three tiers of context management that mirror Claude Code's
// design. Each tier is heavier than the previous one and they are intended to
// be applied as a cascade:
//
//  1. Microcompaction - no LLM; clears bulky old tool results in memory so the
//     hot tail of recent results stays visible while older ones are replaced
//     with a placeholder.
//  2. Session memory compaction - no LLM; reuses a previously stored
//     compaction summary to replace the history. Falls back to a full
//     compaction when the stored summary does not fit below the threshold.
//  3. Auto-compaction - LLM-based structured summarization of the whole
//     history into a working state, then replaces the history with a boundary
//     marker and a continuation message so the agent resumes seamlessly.
//
// All mechanisms operate on eino schema.Message slices and never mutate the
// input; they return new slices instead.
package compact

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Default tuning constants. Values are derived from Claude Code's reverse
// engineered compaction thresholds and can be overridden via Options.
const (
	// DefaultContextWindow is the assumed model context window in tokens.
	DefaultContextWindow = 200_000
	// DefaultReservedOutputTokens caps the tokens reserved for model output
	// (the p99.99 compaction summary is roughly 17.4K tokens).
	DefaultReservedOutputTokens = 20_000
	// DefaultAutoCompactBuffer is the safety buffer below the usable window at
	// which auto-compaction fires.
	DefaultAutoCompactBuffer = 13_000
	// DefaultWarningBuffer is how far below the auto-compact trigger the
	// "context almost full" warning is shown.
	DefaultWarningBuffer = 20_000
	// DefaultBlockingBuffer is the hard limit offset from the context window.
	DefaultBlockingBuffer = 3_000

	// Microcompaction tuning.

	// DefaultProtectedToolResults is the number of most recent tool results
	// always kept intact (the hot tail).
	DefaultProtectedToolResults = 3
	// DefaultProtectedToolResultTokens is the token budget of recent tool
	// results kept before older ones become eligible for clearing.
	DefaultProtectedToolResultTokens = 40_000
	// DefaultMinSavingsTokens is the minimum token savings required for
	// microcompaction to bother clearing anything.
	DefaultMinSavingsTokens = 20_000
)

// toolResultClearedPlaceholder replaces bulky tool results during
// microcompaction.
const toolResultClearedPlaceholder = "[Tool result cleared]"

// Model is the minimal contract required for LLM-driven compaction. The
// signature matches eino's chat model Generate so *openai.ChatModel satisfies
// it, while still allowing lightweight fakes in tests.
type Model interface {
	Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error)
}

// SessionSummaryStore persists the most recent compaction summary so that
// session memory compaction can reuse it without calling the model again.
type SessionSummaryStore interface {
	// GetSessionSummary returns the stored summary and true when one exists.
	GetSessionSummary(ctx context.Context) (summary string, ok bool, err error)
	// SaveSessionSummary stores (or replaces) the current compaction summary.
	SaveSessionSummary(ctx context.Context, summary string) error
}

// Options configures the compaction mechanisms. Zero values are replaced with
// the corresponding defaults at use time.
type Options struct {
	// ContextWindow is the assumed model context window in tokens.
	ContextWindow int
	// ReservedOutputTokens caps the model output reservation. Defaults to 20K.
	ReservedOutputTokens int
	// AutoCompactBuffer is the safety buffer below the usable window.
	// Defaults to 13K.
	AutoCompactBuffer int
	// WarningBuffer is the offset below the auto-compact trigger at which the
	// warning shows. Defaults to 20K.
	WarningBuffer int
	// BlockingBuffer is the hard limit offset from the context window.
	// Defaults to 3K.
	BlockingBuffer int

	// Microcompaction tuning.

	// ProtectedToolResults is the hot tail size (tool results always kept).
	ProtectedToolResults int
	// ProtectedToolResultTokens is the token budget of recent tool results
	// kept before older ones become eligible for clearing.
	ProtectedToolResultTokens int
	// MinSavingsTokens is the minimum savings required for clearing.
	MinSavingsTokens int
	// CompactableTools is the set of tool names whose results may be cleared.
	// Defaults to DefaultCompactableTools.
	CompactableTools map[string]struct{}

	// CustomInstructions is an optional focus hint appended to the
	// summarization prompt (e.g. "focus on the auth bug fix").
	CustomInstructions string

	// SummaryStore, when set, lets auto-compaction persist its summary so that
	// session memory compaction can reuse it later.
	SummaryStore SessionSummaryStore
}

// DefaultOptions returns Options populated with Claude Code style defaults.
func DefaultOptions() Options {
	return Options{
		ContextWindow:             DefaultContextWindow,
		ReservedOutputTokens:      DefaultReservedOutputTokens,
		AutoCompactBuffer:         DefaultAutoCompactBuffer,
		WarningBuffer:             DefaultWarningBuffer,
		BlockingBuffer:            DefaultBlockingBuffer,
		ProtectedToolResults:      DefaultProtectedToolResults,
		ProtectedToolResultTokens: DefaultProtectedToolResultTokens,
		MinSavingsTokens:          DefaultMinSavingsTokens,
		CompactableTools:          DefaultCompactableTools(),
	}
}

// normalized returns a copy of o with zero values replaced by defaults.
func (o Options) normalized() Options {
	if o.ContextWindow <= 0 {
		o.ContextWindow = DefaultContextWindow
	}
	if o.ReservedOutputTokens <= 0 {
		o.ReservedOutputTokens = DefaultReservedOutputTokens
	}
	if o.AutoCompactBuffer <= 0 {
		o.AutoCompactBuffer = DefaultAutoCompactBuffer
	}
	if o.WarningBuffer <= 0 {
		o.WarningBuffer = DefaultWarningBuffer
	}
	if o.BlockingBuffer <= 0 {
		o.BlockingBuffer = DefaultBlockingBuffer
	}
	if o.ProtectedToolResults <= 0 {
		o.ProtectedToolResults = DefaultProtectedToolResults
	}
	if o.ProtectedToolResultTokens <= 0 {
		o.ProtectedToolResultTokens = DefaultProtectedToolResultTokens
	}
	if o.MinSavingsTokens <= 0 {
		o.MinSavingsTokens = DefaultMinSavingsTokens
	}
	if o.CompactableTools == nil {
		o.CompactableTools = DefaultCompactableTools()
	}
	return o
}

// EffectiveWindow is the usable context window after reserving output space.
func (o Options) EffectiveWindow() int {
	o = o.normalized()
	return o.ContextWindow - o.ReservedOutputTokens
}

// AutoCompactThreshold is the token count at which auto-compaction fires.
func (o Options) AutoCompactThreshold() int {
	o = o.normalized()
	return o.EffectiveWindow() - o.AutoCompactBuffer
}

// WarningThreshold is the token count at which the context warning shows.
func (o Options) WarningThreshold() int {
	o = o.normalized()
	return o.AutoCompactThreshold() - o.WarningBuffer
}

// BlockingLimit is the absolute hard stop.
func (o Options) BlockingLimit() int {
	o = o.normalized()
	return o.ContextWindow - o.BlockingBuffer
}

// DefaultCompactableTools returns the tool names whose bulky results may be
// cleared by microcompaction.
func DefaultCompactableTools() map[string]struct{} {
	tools := []string{
		"read", "read_file",
		"bash", "shell",
		"grep", "glob",
		"edit", "write",
		"web_search", "websearch",
		"web_fetch", "webfetch",
	}
	m := make(map[string]struct{}, len(tools))
	for _, t := range tools {
		m[t] = struct{}{}
	}
	return m
}

// Result is the outcome of a compaction pass.
type Result struct {
	// Messages is the compacted message list. When no compaction happened it
	// equals the input.
	Messages []*schema.Message
	// Compacted reports whether any change was applied.
	Compacted bool
	// TokensBefore is the estimated token count of the input.
	TokensBefore int
	// TokensAfter is the estimated token count of the result.
	TokensAfter int
	// TokensSaved is TokensBefore - TokensAfter.
	TokensSaved int
}

func newResult(before []*schema.Message, after []*schema.Message, compacted bool) *Result {
	tb := EstimateTokensForMessages(before)
	ta := EstimateTokensForMessages(after)
	return &Result{
		Messages:     after,
		Compacted:    compacted,
		TokensBefore: tb,
		TokensAfter:  ta,
		TokensSaved:  tb - ta,
	}
}

// EstimateTokens returns a rough token count for s using the character/4
// heuristic Claude Code uses for un-tokenized content.
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	n := utf8.RuneCountInString(s)
	return (n + 3) / 4
}

// EstimateMessageTokens returns an approximate token count for a single
// message, accounting for text content, reasoning, tool calls and tool
// results.
func EstimateMessageTokens(m *schema.Message) int {
	if m == nil {
		return 0
	}
	tokens := EstimateTokens(m.Content)
	tokens += EstimateTokens(m.ReasoningContent)
	for _, tc := range m.ToolCalls {
		tokens += EstimateTokens(tc.Function.Name)
		tokens += EstimateTokens(tc.Function.Arguments)
	}
	tokens += EstimateTokens(m.ToolName)
	return tokens
}

// EstimateTokensForMessages sums the token estimates across a message slice.
func EstimateTokensForMessages(messages []*schema.Message) int {
	var total int
	for _, m := range messages {
		total += EstimateMessageTokens(m)
	}
	return total
}

// ShouldAutoCompact reports whether the estimated token count of messages has
// reached the auto-compaction threshold.
func ShouldAutoCompact(messages []*schema.Message, opts Options) bool {
	opts = opts.normalized()
	return EstimateTokensForMessages(messages) >= opts.AutoCompactThreshold()
}

// IsAboveWarningThreshold reports whether the warning threshold is exceeded.
func IsAboveWarningThreshold(messages []*schema.Message, opts Options) bool {
	opts = opts.normalized()
	return EstimateTokensForMessages(messages) >= opts.WarningThreshold()
}

// isCompactableTool reports whether a tool name belongs to the compactable
// set (case-insensitive).
func isCompactableTool(name string, tools map[string]struct{}) bool {
	if name == "" {
		return false
	}
	_, ok := tools[strings.ToLower(name)]
	return ok
}

// buildContinuationMessage wraps a compaction summary in the continuation
// framing Claude Code injects after compaction so the agent resumes without
// re-asking the user.
func buildContinuationMessage(summary string) *schema.Message {
	var sb strings.Builder
	sb.WriteString("## [Conversation compacted]\n\n")
	sb.WriteString("This session is being continued from a previous conversation that ran ")
	sb.WriteString("out of context. The summary below covers the earlier portion of the ")
	sb.WriteString("conversation.\n\n")
	sb.WriteString(strings.TrimSpace(summary))
	sb.WriteString("\n\nPlease continue the conversation from where we left off without ")
	sb.WriteString("asking the user any further questions. Continue with the last task ")
	sb.WriteString("that you were asked to work on.")
	return &schema.Message{Role: schema.User, Content: sb.String()}
}

// formatConversation renders messages into the "[role]: content" form used in
// the summarization prompt. Tool-call-only assistant messages are rendered
// from their calls so the summary captures what tools were used.
func formatConversation(messages []*schema.Message) string {
	var sb strings.Builder
	for _, m := range messages {
		role := string(m.Role)
		if role == "" {
			role = "user"
		}
		content := m.Content
		if content == "" && len(m.ToolCalls) > 0 {
			content = formatToolCalls(m.ToolCalls)
		}
		sb.WriteString("[" + role + "]: " + content + "\n")
	}
	return sb.String()
}

func formatToolCalls(calls []schema.ToolCall) string {
	var sb strings.Builder
	for i, c := range calls {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString("tool_call ")
		sb.WriteString(c.Function.Name)
		sb.WriteString("(")
		sb.WriteString(c.Function.Arguments)
		sb.WriteString(")")
	}
	return sb.String()
}

// between returns the substring of s delimited by open and close tags and
// reports whether both delimiters were found.
func between(s, open, closeTag string) (string, bool) {
	i := strings.Index(s, open)
	if i < 0 {
		return "", false
	}
	start := i + len(open)
	j := strings.Index(s[start:], closeTag)
	if j < 0 {
		return "", false
	}
	return s[start : start+j], true
}
