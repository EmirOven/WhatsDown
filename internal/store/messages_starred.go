package store

import "context"

func (s *MessageStore) ToggleMessageStarred(ctx context.Context, chatJID, id string, starred bool) error {
	v := 0
	if starred {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE whatsdown_messages
		SET starred = ?
		WHERE chat_jid = ? AND id = ?
	`, v, chatJID, id)
	return err
}

func (s *MessageStore) ListStarredMessages(ctx context.Context, chatJID string) ([]StoredMessage, error) {
	query := selectMessagesSQL + `
		WHERE m.chat_jid = ? AND m.starred = 1
		ORDER BY m.timestamp DESC`
	rows, err := s.db.QueryContext(ctx, query, chatJID)
	if err != nil {
		return nil, err
	}
	return scanStoredMessages(rows)
}
