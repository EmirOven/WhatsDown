package store

const migration16 = `
CREATE TABLE IF NOT EXISTS whatsdown_chat_folders (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 color TEXT NOT NULL,
 position INTEGER NOT NULL,
 collapsed INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS whatsdown_chat_folder_members (
 chat_jid TEXT PRIMARY KEY,
 folder_id TEXT NOT NULL REFERENCES whatsdown_chat_folders(id) ON DELETE CASCADE,
 position INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS whatsdown_folder_members_order ON whatsdown_chat_folder_members(folder_id, position);
`
