package store

const migration9 = `
CREATE TABLE IF NOT EXISTS whatsdown_saved_stickers (
	id          TEXT PRIMARY KEY,
	file_path   TEXT NOT NULL,
	file_hash   TEXT NOT NULL DEFAULT '',
	width       INTEGER NOT NULL DEFAULT 512,
	height      INTEGER NOT NULL DEFAULT 512,
	saved_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS whatsdown_saved_stickers_saved_at ON whatsdown_saved_stickers (saved_at DESC);
`

const migration10 = `
CREATE TABLE IF NOT EXISTS whatsdown_message_edits (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	message_id  TEXT NOT NULL,
	chat_jid    TEXT NOT NULL,
	text        TEXT NOT NULL,
	edited_at   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_whatsdown_message_edits_msg ON whatsdown_message_edits (chat_jid, message_id);
`

const migration11 = `
ALTER TABLE whatsdown_chats ADD COLUMN parent_jid TEXT NOT NULL DEFAULT '';
ALTER TABLE whatsdown_chats ADD COLUMN is_parent INTEGER NOT NULL DEFAULT 0;
ALTER TABLE whatsdown_chats ADD COLUMN is_default_sub_group INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_whatsdown_chats_parent ON whatsdown_chats (parent_jid);
`

const migration12 = `
CREATE TABLE IF NOT EXISTS whatsdown_calls (
	id            TEXT PRIMARY KEY,
	chat_jid      TEXT NOT NULL,
	caller_jid    TEXT NOT NULL,
	caller_name   TEXT NOT NULL DEFAULT '',
	timestamp     INTEGER NOT NULL,
	duration_secs INTEGER NOT NULL DEFAULT 0,
	is_video      INTEGER NOT NULL DEFAULT 0,
	is_group      INTEGER NOT NULL DEFAULT 0,
	outcome       INTEGER NOT NULL DEFAULT 0,
	from_me       INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_whatsdown_calls_time ON whatsdown_calls (timestamp DESC);
`

const migration13 = `
CREATE TABLE IF NOT EXISTS whatsdown_reactions (
	chat_jid    TEXT NOT NULL,
	message_id  TEXT NOT NULL,
	sender_jid  TEXT NOT NULL,
	sender_name TEXT NOT NULL DEFAULT '',
	emoji       TEXT NOT NULL,
	from_me     INTEGER NOT NULL DEFAULT 0,
	timestamp   INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (chat_jid, message_id, sender_jid)
);
CREATE INDEX IF NOT EXISTS idx_whatsdown_reactions_msg ON whatsdown_reactions (chat_jid, message_id);
`
