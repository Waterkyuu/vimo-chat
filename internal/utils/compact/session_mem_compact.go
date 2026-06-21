package compact

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/schema"
)

// ErrSummaryStoreRequired is returned when session memory compaction is
// attempted without a summary store.
var ErrSummaryStoreRequired = errors.New("compact: summary store is required for session memory compaction")

// SessionMemoryCompact reuses a previously stored compaction summary to
// replace the history without calling the model. It mirrors Claude Code's
// session memory compaction, which is tried before a full LLM compaction:
//
//   - If the store has a summary, the history is replaced by the continuation
//     framing carrying that summary (no model call).
//   - The compacted result is only accepted when it still fits below the
//     auto-compaction threshold; otherwise the caller should fall back to a
//     full LLM compaction (AutoCompact).
//
// It returns (result, applied). applied is true only when a stored summary was
// found and the compacted messages fit under the threshold. When applied is
// false and no error occurred, the caller should run AutoCompact. The input
// slice is never mutated.
func SessionMemoryCompact(
	ctx context.Context,
	messages []*schema.Message,
	store SessionSummaryStore,
	opts Options,
) (*Result, bool, error) {
	if store == nil {
		return nil, false, ErrSummaryStoreRequired
	}
	opts = opts.normalized()

	summary, ok, err := store.GetSessionSummary(ctx)
	if err != nil {
		return nil, false, err
	}
	if !ok || summary == "" {
		// Nothing to reuse: caller should run a full compaction.
		return newResult(messages, messages, false), false, nil
	}

	out := []*schema.Message{buildContinuationMessage(summary)}
	result := newResult(messages, out, true)

	// Only accept session memory when the result still fits below the
	// threshold; otherwise signal fallback to LLM compaction.
	if result.TokensAfter >= opts.AutoCompactThreshold() {
		return result, false, nil
	}
	return result, true, nil
}
