package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

const defaultMarkdownWidth = 80

// markdownRenderer converts markdown into ANSI-styled text using glamour, which
// belongs to the same Charm stack as bubbletea/lipgloss and is therefore the
// idiomatic choice for rendering LLM output in this TUI.
//
// The underlying renderer is rebuilt only when the wrap width changes (terminal
// resize), and rendered output is memoized per source string. Re-rendering the
// whole transcript on every streamed token therefore stays cheap: completed
// messages hit the cache while only the actively growing message pays the cost.
type markdownRenderer struct {
	renderer *glamour.TermRenderer
	width    int
	cache    map[string]string
}

func newMarkdownRenderer() *markdownRenderer {
	return &markdownRenderer{cache: make(map[string]string)}
}

// render returns the ANSI-styled representation of content, rebuilding the
// glamour renderer and flushing the cache whenever width changes. On any
// failure it falls back to the raw content so the UI never breaks.
func (r *markdownRenderer) render(content string, width int) string {
	if content == "" {
		return ""
	}

	wrap := width
	if wrap <= 0 {
		wrap = defaultMarkdownWidth
	}

	if r.renderer == nil || r.width != wrap {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(wrap),
		)
		if err != nil {
			return content
		}
		r.renderer = renderer
		r.width = wrap
		r.cache = make(map[string]string)
	}

	if out, ok := r.cache[content]; ok {
		return out
	}

	out, err := r.renderer.Render(content)
	if err != nil {
		return content
	}

	// Glamour pads its output with surrounding blank lines; trim them so the
	// result composes cleanly with the message separators in renderMessage.
	out = strings.TrimSpace(out)
	r.cache[content] = out
	return out
}
