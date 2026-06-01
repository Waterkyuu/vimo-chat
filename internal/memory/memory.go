package memory

import "time"

type MemoryType string

const (
	// MemoryTypeSavedFact: Facts explicitly requested by the user to be remembered (such as "I prefer dark themes")
	MemoryTypeSavedFact MemoryType = "saved_fact"
	// MemoryTypeExtractedKnowledge LLM from the dialogue in the history of automatic extraction of knowledge
	MemoryTypeExtractedKnowledge MemoryType = "extracted_knowledge"
)

// Memory: A single piece of memory
type Memory struct {
	ID             string     `json:"id"`
	Type           MemoryType `json:"type"`
	Content        string     `json:"content"`
	ConversationID string     `json:"conversation_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	UsageCount     int        `json:"usage_count"`
}

// Conversation Record
type Conversation struct {
	ID           string     `json:"id"`
	Title        string     `json:"title,omitempty"`
	Summary      string     `json:"summary,omitempty"`
	MessageCount int        `json:"message_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	SummarizedAt *time.Time `json:"summarized_at,omitempty"`
}

// MemoryFilter: Memory query filtering conditions
type MemoryFilter struct {
	Type   MemoryType
	Limit  int
	Offset int
}
