package store_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"postbetasoftware.com/whatsdown/internal/store"
)

func setupTestStore(t *testing.T) (*store.MessageStore, context.Context, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	s, err := store.OpenMessageStore(ctx, db)
	if err != nil {
		db.Close()
		t.Fatalf("openMessageStore: %v", err)
	}
	return s, ctx, db
}
