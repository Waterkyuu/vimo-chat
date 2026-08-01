package compact

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestSessionMemoryCompact_ErrStoreRequired(t *testing.T) {
	_, _, err := SessionMemoryCompact(context.Background(), nil, nil, DefaultOptions())
	if !errors.Is(err, ErrSummaryStoreRequired) {
		t.Fatalf("err = %v, want ErrSummaryStoreRequired", err)
	}
}

func TestSessionMemoryCompact_NoSummary(t *testing.T) {
	store := &fakeStore{ok: false}
	msgs := []*schema.Message{userMsg("a"), assistantMsg("b")}

	r, applied, err := SessionMemoryCompact(context.Background(), msgs, store, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied {
		t.Fatal("applied should be false when no summary is stored")
	}
	if r.Compacted {
		t.Fatal("Compacted should be false when no summary is stored")
	}
}

func TestSessionMemoryCompact_AppliedWhenFits(t *testing.T) {
	store := &fakeStore{summary: "previous compaction summary", ok: true}
	msgs := []*schema.Message{
		userMsg(strings.Repeat("x", 4000)),
		assistantMsg(strings.Repeat("y", 4000)),
	}

	r, applied, err := SessionMemoryCompact(context.Background(), msgs, store, DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !applied {
		t.Fatal("expected applied=true when the compacted result fits")
	}
	if !r.Compacted {
		t.Fatal("expected Compacted=true")
	}
	if len(r.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(r.Messages))
	}
	if !strings.Contains(r.Messages[0].Content, "previous compaction summary") {
		t.Fatalf("stored summary not in continuation message: %q", r.Messages[0].Content)
	}
}

func TestSessionMemoryCompact_FallbackWhenTooBig(t *testing.T) {
	// Force a tiny context window so the resulting continuation message exceeds
	// the auto-compact threshold, signaling fallback to full compaction. The
	// continuation framing is ~90 tokens, so a window well below that triggers
	// the fallback.
	opts := Options{ContextWindow: 50, ReservedOutputTokens: 1, AutoCompactBuffer: 1}
	store := &fakeStore{summary: "summary", ok: true}
	msgs := []*schema.Message{userMsg("a")}

	r, applied, err := SessionMemoryCompact(context.Background(), msgs, store, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied {
		t.Fatal("expected applied=false when the result does not fit (fallback)")
	}
	if !r.Compacted {
		t.Fatal("Compacted should still be true (history was replaced); only applied signals fallback")
	}
}

func TestSessionMemoryCompact_StoreGetError(t *testing.T) {
	store := &fakeStore{err: errors.New("db down")}
	msgs := []*schema.Message{userMsg("a")}

	_, _, err := SessionMemoryCompact(context.Background(), msgs, store, DefaultOptions())
	if err == nil {
		t.Fatal("expected store error to propagate")
	}
}
