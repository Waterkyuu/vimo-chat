// LLM-driven memory retrieval
package memory

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

type Extractor struct {
	model *openai.ChatModel
}

func NewExtractor(model *openai.ChatModel) *Extractor {
	return &Extractor{model: model}
}

// Extract key facts and knowledge from dialogue messages - Phase 1
func (e *Extractor) ExtractMemoried(ctx context.Context, messages []*schema.Message) ([]*Memory, error) {
	prompt := buildExtractionPrompt(messages)

	res, err := e.model.Generate(ctx, []*schema.Message{
		{
			Role: schema.System, Content: memoryExtractionSystemPrompt,
		},
		{
			Role:    schema.User,
			Content: prompt,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("Extract memoried: %w", err)
	}

	return parseExtractedMemories(res.Content)
}

// Uses LLM to generate short summaries of conversations
func (e *Extractor) SummarizeConversation(ctx context.Context, messages []*schema.Message) (string, error) {
	prompt := buildSummaryPrompt(messages)

	resp, err := e.model.Generate(ctx, []*schema.Message{
		{Role: schema.System, Content: summarySystemPrompt},
		{Role: schema.User, Content: prompt},
	})
	if err != nil {
		return "", fmt.Errorf("summarize conversation: %w", err)
	}

	return resp.Content, nil
}

// Merge duplicates - Phase 2
func (e *Extractor) ConsolidateMemories(ctx context.Context, memories []*Memory) ([]*Memory, error) {
	if len(memories) <= 5 {
		return memories, nil
	}

	prompt := buildConsolidationPrompt(memories)

	resp, err := e.model.Generate(ctx, []*schema.Message{
		{Role: schema.System, Content: consolidationSystemPrompt},
		{Role: schema.User, Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("consolidate memories: %w", err)
	}

	return parseExtractedMemories(resp.Content)
}

// ---- Prompt template ----

// Extractor prommpt template
const memoryExtractionSystemPrompt = `You are a memory extraction assistant. Your task is to identify important information from conversations that should be remembered.

Rules:
- Extract user preferences, habits, constraints, and important facts
- Extract technical preferences (programming languages, frameworks, tools, code styles)
- Extract project-specific knowledge
- Do not extract temporary information (current weather, one-off questions)
- Each memory should be a standalone sentence
- Return as a JSON array, with each item containing a "content" field
- If there is nothing worth remembering, return an empty array []
- Do NOT call any tools or functions under any circumstances

Example output:
[
  {"content": "User prefers dark theme"},
  {"content": "User's primary programming language is Go"}
]`

// Summary prompt template
const summarySystemPrompt = `Summarize this conversation in 2-3 sentences. Focus on:
- What was discussed
- Key decisions or conclusions reached
- Important context that will be valuable for future reference

Keep it concise and objective.`

const consolidationSystemPrompt = `You are a memory consolidation assistant. Given a set of memory entries, merge duplicates, resolve conflicts, and remove outdated information.

Rules:
- Merge different memories that express the same meaning
- When memories conflict, keep the newest/most authoritative one
- Remove trivial or obvious facts
- Each result should be a standalone sentence
- Return as a JSON array, with each item containing a "content" field
- Preserve all non-conflicting unique information`

func buildExtractionPrompt(messages []*schema.Message) string {
	var conversation string
	for _, msg := range messages {
		conversation += fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content)
	}
	return fmt.Sprintf("Extract important memory from the following conversations:\n\n%s", conversation)
}

func buildSummaryPrompt(messages []*schema.Message) string {
	var conversation string
	for _, msg := range messages {
		conversation += fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content)
	}
	return fmt.Sprintf("Summarize the following conversations: \n\n%s", conversation)
}

func buildConsolidationPrompt(memories []*Memory) string {
	var list string
	for i, m := range memories {
		list += fmt.Sprintf("%d. [%s] %s\n", i+1, m.Type, m.Content)
	}
	return fmt.Sprintf("Integrate the following memories and merge duplicates: \n\n%s", list)
}

// ParseExtractedMemories: parses the JSON returned by the LLM into a list of memories
func parseExtractedMemories(content string) ([]*Memory, error) {
	// TODO: Use json.Unmarshal parse `[{"content": "..."}, ...]`
	// Generated UUID, for each type of set as MemoryTypeExtractedKnowledge
	return nil, nil
}
