// LLM-driven memory retrieval
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
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

// parseExtractedMemories parses the JSON array returned by the LLM into a list
// of memories. Each item must contain a "content" field. The LLM may wrap the
// payload in markdown code fences or surround it with prose, so the JSON body
// is isolated before decoding. Each memory gets a fresh UUID and the type
// MemoryTypeExtractedKnowledge; timestamps are left to the caller.
//
// Example input (raw LLM response):
//
//	```json
//	[
//	  {"content": "User prefers Go"},
//	  {"content": "  "},
//	  {"content": "User likes dark theme"}
//	]
//	```
//
// Example output ([]*Memory):
//
//	[]*Memory{
//	    {ID: "a1b2c3d4-...", Type: MemoryTypeExtractedKnowledge, Content: "User prefers Go"},
//	    // The second item is empty after trimming and is skipped.
//	    {ID: "e5f6g7h8-...", Type: MemoryTypeExtractedKnowledge, Content: "User likes dark theme"},
//	}
//
// An empty array ("[]") yields (nil, nil). Invalid JSON yields (nil, error).
func parseExtractedMemories(content string) ([]*Memory, error) {
	body := isolateJSONArray(content)
	if body == "" || body == "[]" {
		return nil, nil
	}

	var items []struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(body), &items); err != nil {
		return nil, fmt.Errorf("parse extracted memories: %w", err)
	}

	memories := make([]*Memory, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(item.Content)
		if text == "" {
			continue
		}
		memories = append(memories, &Memory{
			ID:      uuid.New().String(),
			Type:    MemoryTypeExtractedKnowledge,
			Content: text,
		})
	}

	return memories, nil
}

// isolateJSONArray extracts the JSON array payload from a raw LLM response,
// stripping markdown code fences (```json ... ```) and any leading/trailing
// prose by locating the outermost [ ... ] boundaries.
//
// Example input 1 (markdown code fences):
//
//	```json
//	[{"content": "User uses Vim"}]
//	```
//
// Example output 1:
//
//	[{"content": "User uses Vim"}]
//
// Example input 2 (surrounding prose):
//
//	Here are the memories:
//	[{"content": "User uses Vim"}]
//	Hope this helps!
//
// Example output 2:
//
//	[{"content": "User uses Vim"}]
//
// Example input 3 (empty array):
//
//	[]
//
// Example output 3:
//
//	[]
func isolateJSONArray(raw string) string {
	s := strings.TrimSpace(raw)

	// Strip markdown code fences: ```json\n...\n``` or ```\n...\n```.
	if strings.HasPrefix(s, "```") {
		if nl := strings.Index(s, "\n"); nl != -1 {
			s = s[nl+1:]
		}
		if i := strings.LastIndex(s, "```"); i != -1 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}

	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}

	return s
}
