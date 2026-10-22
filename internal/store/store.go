package store

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// MessageStore handles chat, message, call, reaction, and poll persistence in SQLite.
type MessageStore struct {
	db                *sql.DB
	demo              bool
	selfMu            sync.RWMutex
	selfPN            string
	selfLID           string
	stickerSyncMu     sync.Mutex
	OnStickersUpdated func()
}

// SetDemo marks whether this store is running in Demo Mode.
func (s *MessageStore) SetDemo(demo bool) {
	if s != nil {
		s.demo = demo
	}
}

// IsDemo returns true if this store is running in Demo Mode.
func (s *MessageStore) IsDemo() bool {
	return s != nil && s.demo
}

// execer is satisfied by both *sql.DB and *sql.Tx.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// OpenMessageStore initializes the database schema and performs migrations.
func OpenMessageStore(ctx context.Context, db *sql.DB) (*MessageStore, error) {
	s := &MessageStore{db: db}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrating message store: %w", err)
	}
	s.cleanupSelfChats(ctx)
	s.cleanupDuplicateChats(ctx)
	return s, nil
}

// DB returns the underlying sql.DB.
func (s *MessageStore) DB() *sql.DB {
	return s.db
}
