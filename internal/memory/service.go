package memory

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino-ext/components/model/openai"
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