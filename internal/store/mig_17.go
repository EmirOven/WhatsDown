package store

const migration17 = `
CREATE TABLE IF NOT EXISTS whatsdown_message_recipients (
 chat_jid TEXT NOT NULL,
 message_id TEXT NOT NULL,
 recipient_jid TEXT NOT NULL,
 expected INTEGER NOT NULL DEFAULT 0,
 status INTEGER NOT NULL DEFAULT 1,
 delivered_at INTEGER NOT NULL DEFAULT 0,
 read_at INTEGER NOT NULL DEFAULT 0,
 PRIMARY KEY (chat_jid, message_id, recipient_jid)
);
CREATE TABLE IF NOT EXISTS whatsdown_message_recipient_sets (
 chat_jid TEXT NOT NULL,
 message_id TEXT NOT NULL,
 PRIMARY KEY (chat_jid, message_id)
);
UPDATE whatsdown_messages SET status = 1 WHERE from_me = 1 AND chat_jid LIKE '%@g.us';
`
