package memory

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Store interface {
	// Memory CRUD
	SaveMemory(ctx context.Context, m *Memory) error
	DeleteMemory(ctx context.Context, id string) error
	GetMemory(ctx context.Context, id string) (*Memory, error)
	ListMemories(ctx context.Context, filter MemoryFilter) ([]*Memory, error)
	SearchMemories(ctx context.Context, query string) ([]*Memory, error)
	RecordUsage(ctx context.Context, ids []string) error

	// Dialogue CRUD
	SaveConversation(ctx context.Context, c *Conversation) error
	GetConversation(ctx context.Context, id string) (*Conversation, error)
	GetStaleConversations(ctx context.Context, limit int) ([]*Conversation, error)
	DeleteConversation(ctx context.Context, id string) error

	Close() error
}

// SQLiteStore is a storage implementation based on SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates an SQLite storage instance and automatically executes migration
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// 开启 WAL 模式提升并发读性能
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
        CREATE TABLE IF NOT EXISTS memories (
            id              TEXT PRIMARY KEY,
            type            TEXT NOT NULL,
            content         TEXT NOT NULL,
            conversation_id TEXT,
            created_at      DATETIME NOT NULL,
            updated_at      DATETIME NOT NULL,
            last_used_at    DATETIME,
            usage_count     INTEGER DEFAULT 0
        );
        CREATE INDEX IF NOT EXISTS idx_memories_type ON memories(type);
        CREATE INDEX IF NOT EXISTS idx_memories_last_used ON memories(last_used_at);

        CREATE TABLE IF NOT EXISTS conversations (
            id              TEXT PRIMARY KEY,
            title           TEXT,
            summary         TEXT,
            message_count   INTEGER DEFAULT 0,
            created_at      DATETIME NOT NULL,
            updated_at      DATETIME NOT NULL,
            summarized_at   DATETIME
        );
        CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at);
    `)
	return err
}

// SaveMemory
func (s *SQLiteStore) SaveMemory(ctx context.Context, m *Memory) error {
	_, err := s.db.ExecContext(ctx, `
        INSERT OR REPLACE INTO memories (id, type, content, conversation_id, created_at, updated_at, last_used_at, usage_count)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `, m.ID, m.Type, m.Content, m.ConversationID, m.CreatedAt, m.UpdatedAt, m.LastUsedAt, m.UsageCount)
	return err
}

// DeleteMemory
func (s *SQLiteStore) DeleteMemory(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM memories WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) GetMemory(ctx context.Context, id string) (*Memory, error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT id, type, content, conversation_id, created_at, updated_at, last_used_at, usage_count
        FROM memories WHERE id = ?
    `, id)
	return scanMemory(row)
}

// ListMemories lists memories based on filtering conditions
func (s *SQLiteStore) ListMemories(ctx context.Context, filter MemoryFilter) ([]*Memory, error) {
	query := `SELECT id, type, content, conversation_id, created_at, updated_at, last_used_at, usage_count
              FROM memories WHERE 1=1`
	var args []any

	if filter.Type != "" {
		query += ` AND type = ?`
		args = append(args, filter.Type)
	}

	query += ` ORDER BY updated_at DESC`

	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		m, err := scanMemoryRow(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

// SearchMemories: Search for memories by keywords (LIKE fuzzy matching)
func (s *SQLiteStore) SearchMemories(ctx context.Context, query string) ([]*Memory, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, type, content, conversation_id, created_at, updated_at, last_used_at, usage_count
        FROM memories WHERE content LIKE ?
        ORDER BY usage_count DESC, updated_at DESC
    `, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		m, err := scanMemoryRow(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

// RecordUsage records memory is used, increments usage_count and updates last_used_at
func (s *SQLiteStore) RecordUsage(ctx context.Context, ids []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	for _, id := range ids {
		_, err := tx.ExecContext(ctx, `
            UPDATE memories SET usage_count = usage_count + 1, last_used_at = ?
            WHERE id = ?
        `, now, id)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SaveConversation saves conversation metadata
func (s *SQLiteStore) SaveConversation(ctx context.Context, c *Conversation) error {
	_, err := s.db.ExecContext(ctx, `
        INSERT OR REPLACE INTO conversations (id, title, summary, message_count, created_at, updated_at, summarized_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, c.ID, c.Title, c.Summary, c.MessageCount, c.CreatedAt, c.UpdatedAt, c.SummarizedAt)
	return err
}

// GetConversation retrieves the conversation based on the ID
func (s *SQLiteStore) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT id, title, summary, message_count, created_at, updated_at, summarized_at
        FROM conversations WHERE id = ?
    `, id)
	var c Conversation
	err := row.Scan(&c.ID, &c.Title, &c.Summary, &c.MessageCount, &c.CreatedAt, &c.UpdatedAt, &c.SummarizedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

// GetStaleConversations retrieves conversations that have not yet generated summaries (i.e., "expired" ones pending processing)
func (s *SQLiteStore) GetStaleConversations(ctx context.Context, limit int) ([]*Conversation, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT id, title, summary, message_count, created_at, updated_at, summarized_at
        FROM conversations
        WHERE summarized_at IS NULL OR updated_at > summarized_at
        ORDER BY updated_at DESC
        LIMIT ?
    `, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []*Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(
			&c.ID,
			&c.Title,
			&c.Summary,
			&c.MessageCount,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.SummarizedAt,
		); err != nil {
			return nil, err
		}
		convs = append(convs, &c)
	}
	return convs, rows.Err()
}

// DeleteConversation
func (s *SQLiteStore) DeleteConversation(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, id)
	return err
}

// Close
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// helper
func scanMemory(row *sql.Row) (*Memory, error) {
	var m Memory
	err := row.Scan(
		&m.ID,
		&m.Type,
		&m.Content,
		&m.ConversationID,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.LastUsedAt,
		&m.UsageCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &m, err
}

func scanMemoryRow(rows *sql.Rows) (*Memory, error) {
	var m Memory
	err := rows.Scan(
		&m.ID,
		&m.Type,
		&m.Content,
		&m.ConversationID,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.LastUsedAt,
		&m.UsageCount,
	)
	return &m, err
}
