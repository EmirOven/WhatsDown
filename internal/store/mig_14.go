package store

const migration14 = `
CREATE TABLE IF NOT EXISTS whatsdown_chat_wallpapers (
	chat_jid    TEXT PRIMARY KEY,
	kind        TEXT NOT NULL DEFAULT 'none',
	value       TEXT NOT NULL DEFAULT '',
	updated_at  INTEGER NOT NULL DEFAULT 0
);
`
