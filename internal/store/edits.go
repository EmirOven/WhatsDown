package store

import (
	"context"
	"time"
)

func (s *MessageStore) SaveMessageEdit(ctx context.Context, chatJID, messageID, newText string) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsdown_message_edits (message_id, chat_jid, text, edited_at)
		VALUES (?, ?, ?, ?)`,
		messageID, chatJID, newText, now)
	return err
}

func (s *MessageStore) GetMessageEdits(ctx context.Context, chatJID, messageID string) ([]MessageEdit, error) {
	query := `
		SELECT id, message_id, chat_jid, text, edited_at
		FROM whatsdown_message_edits
		WHERE chat_jid = ? AND message_id = ?
		ORDER BY edited_at ASC`
	rows, err := s.db.QueryContext(ctx, query, chatJID, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edits []MessageEdit
	for rows.Next() {
		var e MessageEdit
		var ts int64
		if err := rows.Scan(&e.ID, &e.MessageID, &e.ChatJID, &e.Text, &ts); err != nil {
			return nil, err
		}
		e.EditedAt = time.Unix(ts, 0)
		edits = append(edits, e)
	}
	return edits, rows.Err()
}
