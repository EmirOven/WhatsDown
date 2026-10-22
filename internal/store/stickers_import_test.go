package store_test

import (
	"runtime"
	"testing"
)

func TestHistoryStickerScanDoesNotLoadPayloads(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	// Use a minimal table to isolate this query from unrelated store behavior.
	if _, err := db.ExecContext(ctx, `DROP TABLE whatsdown_messages;
 CREATE TABLE whatsdown_messages (id INTEGER, chat_jid TEXT, kind TEXT, raw BLOB);
 INSERT INTO whatsdown_messages VALUES (0, 'chat', 'text', zeroblob(262144));
 INSERT INTO whatsdown_messages VALUES (1, 'chat', 'sticker', NULL);`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		if _, err := db.ExecContext(ctx, `INSERT INTO whatsdown_messages VALUES (?, 'chat', 'sticker', zeroblob(262144))`, i+2); err != nil {
			t.Fatal(err)
		}
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	count := s.ImportStickersFromChatHistory(ctx, nil)
	runtime.ReadMemStats(&after)
	if count != 32 {
		t.Fatalf("count = %d, want 32", count)
	}
	// The old SELECT raw + slice allocated at least 8 MiB for this fixture.
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 1<<20 {
		t.Fatalf("sticker scan allocated %d bytes for payloads it does not use", allocated)
	}
}
