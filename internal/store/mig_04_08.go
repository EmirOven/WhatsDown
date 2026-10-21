package store

const migration4 = `
CREATE TABLE IF NOT EXISTS whatsdown_poll_votes (
	chat_jid    TEXT NOT NULL,
	message_id  TEXT NOT NULL,
	option_name TEXT NOT NULL,
	PRIMARY KEY (chat_jid, message_id, option_name)
);
`

const migration5 = `
CREATE TABLE IF NOT EXISTS whatsdown_poll_votes_new (
	chat_jid    TEXT NOT NULL,
	message_id  TEXT NOT NULL,
	voter_jid   TEXT NOT NULL,
	option_name TEXT NOT NULL,
	PRIMARY KEY (chat_jid, message_id, voter_jid, option_name)
);
INSERT OR IGNORE INTO whatsdown_poll_votes_new (chat_jid, message_id, voter_jid, option_name)
	SELECT chat_jid, message_id, 'me', option_name FROM whatsdown_poll_votes;
DROP TABLE whatsdown_poll_votes;
ALTER TABLE whatsdown_poll_votes_new RENAME TO whatsdown_poll_votes;
CREATE INDEX IF NOT EXISTS idx_poll_votes_msg ON whatsdown_poll_votes (chat_jid, message_id);
`

const migration6 = `
ALTER TABLE whatsdown_chats ADD COLUMN muted INTEGER NOT NULL DEFAULT 0;
`

const migration7 = `
ALTER TABLE whatsdown_messages ADD COLUMN status INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS whatsdown_messages_status ON whatsdown_messages (id, from_me);
`

const migration8 = `
UPDATE whatsdown_messages SET status = 3 WHERE from_me = 1 AND status = 0;
`
