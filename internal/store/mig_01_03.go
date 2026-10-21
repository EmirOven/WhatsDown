package store

const migration1 = `
CREATE TABLE whatsdown_chats (
	jid             TEXT PRIMARY KEY,
	name            TEXT    NOT NULL DEFAULT '',
	last_message_at INTEGER NOT NULL DEFAULT 0,
	unread_count    INTEGER NOT NULL DEFAULT 0,
	archived        INTEGER NOT NULL DEFAULT 0,
	pinned          INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE whatsdown_messages (
	chat_jid    TEXT    NOT NULL REFERENCES whatsdown_chats(jid) ON DELETE CASCADE,
	id          TEXT    NOT NULL,
	sender_jid  TEXT    NOT NULL,
	sender_name TEXT    NOT NULL DEFAULT '',
	from_me     INTEGER NOT NULL,
	timestamp   INTEGER NOT NULL,
	kind        TEXT    NOT NULL,
	text        TEXT    NOT NULL DEFAULT '',
	edited      INTEGER NOT NULL DEFAULT 0,
	raw         BLOB,
	PRIMARY KEY (chat_jid, id)
);

CREATE INDEX whatsdown_messages_chat_time ON whatsdown_messages (chat_jid, timestamp);
`

const migration2 = `
CREATE TABLE IF NOT EXISTS whatsmeow_contacts (
	our_jid        TEXT NOT NULL DEFAULT '',
	their_jid      TEXT NOT NULL DEFAULT '',
	first_name     TEXT NOT NULL DEFAULT '',
	full_name      TEXT NOT NULL DEFAULT '',
	push_name      TEXT NOT NULL DEFAULT '',
	business_name  TEXT NOT NULL DEFAULT '',
	redacted_phone TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (our_jid, their_jid)
);

CREATE TABLE IF NOT EXISTS whatsmeow_lid_map (
	lid TEXT PRIMARY KEY,
	pn  TEXT
);
`

const migration3 = `
ALTER TABLE whatsdown_messages ADD COLUMN starred INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS whatsdown_messages_starred ON whatsdown_messages (chat_jid, starred);
`
