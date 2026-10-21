package store

import (
	"context"
	"database/sql"
)

// GetAppSetting returns the string value for a setting key, or defaultVal if not found.
func (s *MessageStore) GetAppSetting(ctx context.Context, key, defaultVal string) (string, error) {
	var val string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM whatsdown_settings WHERE key = ?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return defaultVal, nil
	}
	if err != nil {
		return defaultVal, err
	}
	return val, nil
}

// SetAppSetting sets or updates a setting key-value pair.
func (s *MessageStore) SetAppSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsdown_settings (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}
