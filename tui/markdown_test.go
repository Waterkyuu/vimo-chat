package tui

import (
	"strings"
	"testing"
)

func TestMarkdownRenderEmpty(t *testing.T) {
	r := newMarkdownRenderer()
	if got := r.render("", 80); got != "" {
		t.Fatalf("render(\"\") = %q, want empty", got)
	}
}

func TestMarkdownRenderProducesStyledOutput(t *testing.T) {
	r := newMarkdownRenderer()
	out := r.render("# Hello\n\nSome **bold** text.", 80)

	if out == "" {
		t.Fatal("render returned empty output for valid markdown")
	}
	if !strings.Contains(out, "Hello") {
		t.Fatalf("rendered output dropped heading text:\n%s", out)
	}
	// Glamour wraps content in styled blocks, so the rendered form must differ
	// from the raw markdown source.
	if out == "# Hello\n\nSome **bold** text." {
		t.Fatalf("render returned the raw source unchanged; markdown was not styled:\n%s", out)
	}
}

func TestMarkdownRenderIsDeterministic(t *testing.T) {
	r := newMarkdownRenderer()
	content := "- one\n- two\n- three\n"
	first := r.render(content, 80)
	second := r.render(content, 80)

	if first != second {
		t.Fatalf("render is not deterministic across calls for the same width")
	}
}

func TestMarkdownRenderRebuildsOnWidthChange(t *testing.T) {
	r := newMarkdownRenderer()
	content := "plain paragraph that should wrap"

	narrow := r.render(content, 20)
	wide := r.render(content, 100)

	if narrow == "" || wide == "" {
		t.Fatal("render returned empty output after width change")
	}
	// Changing the wrap width must rebuild the renderer; both results must
	// still contain the source text and remain valid.
	if !strings.Contains(narrow, "wrap") || !strings.Contains(wide, "wrap") {
		t.Fatalf("rendered output dropped source text after width change")
	}
}

func TestMarkdownRenderFallsBackForInvalidWidth(t *testing.T) {
	r := newMarkdownRenderer()
	out := r.render("some content", 0)
	if !strings.Contains(out, "some content") {
		t.Fatalf("render with zero width should fall back and keep source text, got:\n%s", out)
	}
}
