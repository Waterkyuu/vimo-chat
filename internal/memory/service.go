package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/google/uuid"
)

type Service struct {
	store     Store
	extractor *Extractor
}

func NewMemoryService(model *openai.ChatModel) (*Service, error) {
	dbPath, err := defaultDBPath()

	if err != nil {
		return nil, err
	}

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("Init memory store: %w", err)
	}

	return &Service{
		store:     store,
		extractor: NewExtractor(model),
	}, nil

}

// Return the default database path
func defaultDBPath() (string, error) {
	dir, err := os.UserConfigDir()

	if err != nil {
		return "", err
	}

	dbDir := filepath.Join(dir, "vimo")

	if err := os.MkdirAll(dbDir, 0o750); err != nil {
		return "", err
	}

	return filepath.Join(dbDir, "memory.db"), nil
}

// Save a fact that the user display requests to remember
func (s *Service) Remember(ctx context.Context, content string) (*Memory, error) {
	now := time.Now()

	m := &Memory{
		ID:        uuid.New().String(),
		Type:      MemoryTypeSavedFact,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return m, s.store.SaveMemory(ctx, m)
}

// Delete a memory
func (s *Service) Forget(ctx context.Context, id string) error {
	return s.store.DeleteMemory(ctx, id)
}

// Return all memories
func (s *Service) ListAll(ctx context.Context) ([]*Memory, error) {
	return s.store.ListMemories(ctx, MemoryFilter{})
}

// Search for memory by keywords
func (s *Service) Search(ctx context.Context, query string) ([]*Memory, error) {
	return s.store.SearchMemories(ctx, query)
}

// Extract memory and inject system prompt words
func (s *Service) GetContextForPrompt(ctx context.Context) (string, error) {
	memories, err := s.store.ListMemories(ctx, MemoryFilter{})
	if err != nil {
		return "", err
	}

	if len(memories) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("## Memories\n")

	sb.WriteString("The following is about the user's persistent memory for personalized responses. \n\n")

	savedFacts := filterByType(memories, MemoryTypeSavedFact)
	extracted := filterByType(memories, MemoryTypeExtractedKnowledge)

	if len(savedFacts) > 0 {
		sb.WriteString("### Saved facts\n")
		for _, m := range savedFacts {
			sb.WriteString(fmt.Sprintf("- %s\n", m.Content))
		}
		sb.WriteString("\n")
	}

	if len(extracted) > 0 {
		sb.WriteString("### The knowledge learned\n")

		for _, m := range extracted {
			sb.WriteString(fmt.Sprintf("- %s\n", m.Content))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// Close the service and release the database connection
func (s *Service) Close() error {
	return s.store.Close()
}

func filterByType(memories []*Memory, t MemoryType) []*Memory {
	var result []*Memory
	for _, m := range memories {
		if m.Type == t {
			result = append(result, m)
		}
	}

	return result
}
