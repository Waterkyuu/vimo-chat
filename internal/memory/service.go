package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
