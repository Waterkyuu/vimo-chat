package compact

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// Sentinel errors for auto-compaction.
var (
	// ErrModelRequired is returned when auto-compaction is attempted without a
	// configured model.
	ErrModelRequired = errors.New("compact: model is required for auto-compaction")
	// ErrNotEnoughMessages is returned when there is too little history to
	// justify compaction.
	ErrNotEnoughMessages = errors.New("compact: not enough messages to compact")
	// ErrSummaryEmpty is returned when the model returns an empty summary.
	ErrSummaryEmpty = errors.New("compact: failed to generate conversation summary")
)

// autoCompactSystemPrompt is the dedicated summarization system prompt, kept
// deliberately generic so the summarization request can share the main
// conversation's prompt prefix for cache friendliness.
const autoCompactSystemPrompt = "You are a helpful AI assistant tasked with summarizing conversations."

// AutoCompact summarizes the full conversation history into a structured
// working state using the model, then replaces the history with a boundary
// marker and a continuation message. It mirrors Claude Code's auto-compaction:
//
//   - Microcompaction runs first to shed bulky tool results before the model
//     call.
//   - The model is called with NO tools and a structured 9-section prompt that
//     preserves intent, decisions, errors, files, user messages and the next
//     step.
//   - The result is wrapped in the continuation framing so the agent resumes
//     without re-asking the user.
//   - When Options.SummaryStore is set, the generated summary is saved so
//     session memory compaction can reuse it on the next trigger.
//
// model is required. The input slice is never mutated.
func AutoCompact(
	ctx context.Context,
	messages []*schema.Message,
	model Model,
	opts Options,
) (*Result, error) {
	if model == nil {
		return nil, ErrModelRequired
	}
	if len(messages) < 2 {
		return nil, ErrNotEnoughMessages
	}
	opts = opts.normalized()

	// 1. Reduce tokens before summarization via microcompaction.
	micro := MicroCompact(messages, opts)
	input := micro.Messages

	// 2. Ask the model for a structured summary (no tools).
	summary, err := summarize(ctx, input, model, opts.CustomInstructions)
	if err != nil {
		return nil, err
	}

	// 3. Persist the summary for future session memory compaction.
	if opts.SummaryStore != nil {
		if err := opts.SummaryStore.SaveSessionSummary(ctx, summary); err != nil {
			return nil, fmt.Errorf("compact: save session summary: %w", err)
		}
	}

	out := []*schema.Message{buildContinuationMessage(summary)}
	return newResult(messages, out, true), nil
}

// summarize builds the summarization request from the conversation history and
// the structured prompt, then extracts the summary from the model response.
// System messages are dropped from the summarized history because the
// summarization call uses its own dedicated system prompt.
func summarize(
	ctx context.Context,
	messages []*schema.Message,
	model Model,
	customInstructions string,
) (string, error) {
	req := make([]*schema.Message, 0, len(messages)+2)
	req = append(req, &schema.Message{Role: schema.System, Content: autoCompactSystemPrompt})
	for _, m := range messages {
		if m.Role == schema.System {
			continue
		}
		req = append(req, m)
	}
	req = append(req, &schema.Message{Role: schema.User, Content: buildSummaryPrompt(customInstructions)})

	resp, err := model.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("compact: summarize: %w", err)
	}

	summary := extractSummary(resp)
	if strings.TrimSpace(summary) == "" {
		return "", ErrSummaryEmpty
	}
	return summary, nil
}

// extractSummary isolates the final summary from the model response. It prefers
// an explicit <summary> block, otherwise drops any <analysis> scratchpad and
// returns the remaining text trimmed.
func extractSummary(resp *schema.Message) string {
	if resp == nil {
		return ""
	}
	content := resp.Content

	if inner, ok := between(content, "<summary>", "</summary>"); ok {
		return strings.TrimSpace(inner)
	}

	if i := strings.Index(content, "<analysis>"); i >= 0 {
		if j := strings.Index(content, "</analysis>"); j >= 0 {
			content = strings.TrimSpace(content[:i] + content[j+len("</analysis>"):])
		}
	}
	return strings.TrimSpace(content)
}

// buildSummaryPrompt is the structured summarization prompt derived from Claude
// Code's compaction contract. It asks for a reconstruction-grade working state
// organized into nine sections and forbids tool use so the summary call cannot
// trigger slow tool streams.
func buildSummaryPrompt(customInstructions string) string {
	var sb strings.Builder
	sb.WriteString("Your task is to create a detailed summary of the conversation so far, ")
	sb.WriteString("paying close attention to the user's explicit requests and your previous actions. ")
	sb.WriteString("This summary should be thorough in capturing technical details, code patterns, ")
	sb.WriteString("and architectural decisions that would be essential for continuing development work ")
	sb.WriteString("without losing context.\n\n")
	sb.WriteString("Before providing your final summary, wrap your analysis in <analysis> tags to organize ")
	sb.WriteString("your thoughts and ensure you've covered all necessary points. In your analysis:\n\n")
	sb.WriteString("1. Chronologically analyze each message and section of the conversation, identifying the ")
	sb.WriteString("user's explicit requests and intents, your approach, key decisions, technical concepts and ")
	sb.WriteString("code patterns, specific details (file names, full code snippets, function signatures, file ")
	sb.WriteString("edits), errors and how you fixed them, and any user feedback telling you to do something ")
	sb.WriteString("differently.\n")
	sb.WriteString("2. Double-check for technical accuracy and completeness.\n\n")
	sb.WriteString("Then provide your final summary in a <summary> tag with the following sections:\n\n")
	sb.WriteString(
		"1. Primary Request and Intent: Capture all of the user's explicit requests and intents in detail.\n",
	)
	sb.WriteString(
		"2. Key Technical Concepts: List all important technical concepts, technologies, and frameworks discussed.\n",
	)
	sb.WriteString(
		"3. Files and Code Sections: Enumerate specific files and code sections examined, modified, or created. ",
	)
	sb.WriteString(
		"Include full code snippets where applicable and why each matters. Pay special attention to the most ",
	)
	sb.WriteString("recent messages.\n")
	sb.WriteString(
		"4. Errors and fixes: List all errors you ran into, how you fixed them, and any user feedback telling ",
	)
	sb.WriteString("you to do something differently.\n")
	sb.WriteString("5. Problem Solving: Document problems solved and any ongoing troubleshooting efforts.\n")
	sb.WriteString("6. All user messages: List ALL user messages that are not tool results. These are critical for ")
	sb.WriteString("understanding the user's feedback and changing intent.\n")
	sb.WriteString("7. Pending Tasks: Outline any pending tasks that you have explicitly been asked to work on.\n")
	sb.WriteString("8. Current Work: Describe in detail precisely what was being worked on immediately before this ")
	sb.WriteString("summary request, including file names and code snippets where applicable.\n")
	sb.WriteString("9. Optional Next Step: List the next step that is DIRECTLY in line with the user's most recent ")
	sb.WriteString("explicit request and the task you were working on immediately before this summary. If there is a ")
	sb.WriteString(
		"next step, include direct quotes showing exactly what you were working on and where you left off.\n\n",
	)
	if strings.TrimSpace(customInstructions) != "" {
		sb.WriteString("Additional Instructions:\n")
		sb.WriteString(strings.TrimSpace(customInstructions))
		sb.WriteString("\n\n")
	}
	sb.WriteString("IMPORTANT: Do NOT use any tools. You MUST respond with ONLY the <analysis> and <summary> blocks ")
	sb.WriteString("as your text output.")
	return sb.String()
}
