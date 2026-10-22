package store

const migration15 = `
CREATE TABLE IF NOT EXISTS whatsdown_settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL DEFAULT ''
);
`
