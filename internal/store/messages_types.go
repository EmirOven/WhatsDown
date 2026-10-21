package store

import "context"

func (s *MessageStore) ListMediaMessages(ctx context.Context, chatJID string) ([]StoredMessage, error) {
	query := selectMessagesSQL + `
		WHERE m.chat_jid = ? AND m.kind IN ('image', 'video')
		ORDER BY m.timestamp DESC`
	rows, err := s.db.QueryContext(ctx, query, chatJID)
	if err != nil {
		return nil, err
	}
	return scanStoredMessages(rows)
}

func (s *MessageStore) ListAudioMessages(ctx context.Context, chatJID string) ([]StoredMessage, error) {
	query := selectMessagesSQL + `
		WHERE m.chat_jid = ? AND m.kind IN ('audio', 'voice')
		ORDER BY m.timestamp DESC`
	rows, err := s.db.QueryContext(ctx, query, chatJID)
	if err != nil {
		return nil, err
	}
	return scanStoredMessages(rows)
}

func (s *MessageStore) ListDocMessages(ctx context.Context, chatJID string) ([]StoredMessage, error) {
	query := selectMessagesSQL + `
		WHERE m.chat_jid = ? AND m.kind = 'document'
		ORDER BY m.timestamp DESC`
	rows, err := s.db.QueryContext(ctx, query, chatJID)
	if err != nil {
		return nil, err
	}
	return scanStoredMessages(rows)
}
