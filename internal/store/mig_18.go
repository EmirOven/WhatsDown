package store

// Read watermarks survive late message delivery and history replay.
const migration18 = `
CREATE TABLE IF NOT EXISTS whatsdown_chat_reads (
 chat_jid TEXT PRIMARY KEY,
 read_at INTEGER NOT NULL DEFAULT 0
);
`
