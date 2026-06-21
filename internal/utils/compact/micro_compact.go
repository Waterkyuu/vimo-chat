package compact

import (
	"github.com/cloudwego/eino/schema"
)

// MicroCompact clears bulky old tool results in memory without calling the
// model. It mirrors Claude Code's microcompaction, which runs inline before
// each model call to shed context weight cheaply:
//
//   - The most recent ProtectedToolResults (default 3) tool results are always
//     kept intact (the "hot tail") so recent reasoning stays visible.
//   - Going further back, tool results are kept until their cumulative token
//     estimate exceeds ProtectedToolResultTokens (default 40K).
//   - Everything older than that window is eligible for clearing.
//   - Clearing only happens when the eligible tokens exceed MinSavingsTokens
//     (default 20K), so small sessions are left untouched.
//
// Only tool results produced by tools listed in Options.CompactableTools are
// considered (Read, Bash, Grep, Glob, Edit, Write, Web* by default). Cleared
// results keep their role and metadata but have their content replaced with
// toolResultClearedPlaceholder. The input slice is never mutated.
func MicroCompact(messages []*schema.Message, opts Options) *Result {
	opts = opts.normalized()

	// Indices of compactable tool-result messages, in conversation order.
	toolResults := collectCompactableToolResults(messages, opts.CompactableTools)
	if len(toolResults) == 0 {
		return newResult(messages, messages, false)
	}

	protectedTail := opts.ProtectedToolResults
	if protectedTail > len(toolResults) {
		protectedTail = len(toolResults)
	}

	// Walk backwards from just before the protected tail, accumulating tokens
	// until the protection window is exceeded. Everything at or before that
	// boundary is eligible for clearing.
	clearBoundary := -1
	cumulative := 0
	for i := len(toolResults) - protectedTail - 1; i >= 0; i-- {
		cumulative += EstimateMessageTokens(messages[toolResults[i]])
		if cumulative > opts.ProtectedToolResultTokens {
			clearBoundary = i
			break
		}
	}

	eligible := toolResults[:0:0] // nil slice; reassigned below
	if clearBoundary >= 0 {
		eligible = make([]int, 0, clearBoundary+1)
		for i := 0; i <= clearBoundary; i++ {
			eligible = append(eligible, toolResults[i])
		}
	}

	if len(eligible) == 0 {
		return newResult(messages, messages, false)
	}

	var eligibleTokens int
	for _, idx := range eligible {
		eligibleTokens += EstimateMessageTokens(messages[idx])
	}
	if eligibleTokens < opts.MinSavingsTokens {
		return newResult(messages, messages, false)
	}

	out := applyMicroCompact(messages, eligible)
	return newResult(messages, out, true)
}

// collectCompactableToolResults returns the message indices whose role is Tool
// and whose originating tool name belongs to the compactable set.
func collectCompactableToolResults(messages []*schema.Message, tools map[string]struct{}) []int {
	var indices []int
	for i, m := range messages {
		if m != nil && m.Role == schema.Tool && isCompactableTool(m.ToolName, tools) {
			indices = append(indices, i)
		}
	}
	return indices
}

// applyMicroCompact returns a copy of messages with the tool results at the
// given indices replaced by cleared placeholders. Non-cleared messages are
// shared by reference with the input.
func applyMicroCompact(messages []*schema.Message, clearIndices []int) []*schema.Message {
	out := make([]*schema.Message, len(messages))
	clearSet := make(map[int]struct{}, len(clearIndices))
	for _, idx := range clearIndices {
		clearSet[idx] = struct{}{}
	}
	for i, m := range messages {
		if _, ok := clearSet[i]; ok {
			cleared := *m
			cleared.Content = toolResultClearedPlaceholder
			out[i] = &cleared
			continue
		}
		out[i] = m
	}
	return out
}
