package store

import (
	"context"
	"time"
)

func (s *MessageStore) GetStats(ctx context.Context) (chats int, messages int, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM whatsdown_chats`).Scan(&chats)
	if err != nil {
		return
	}
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM whatsdown_messages`).Scan(&messages)
	return
}

func (s *MessageStore) ListMessages(ctx context.Context, chatJID string, before time.Time, limit int) ([]StoredMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	timeCond := ""
	args := []any{chatJID}
	if !before.IsZero() {
		timeCond = " AND m.timestamp < ?"
		args = append(args, before.Unix())
	}
	args = append(args, limit)

	query := selectMessagesSQL + `
		WHERE m.chat_jid = ?` + timeCond + `
		ORDER BY m.timestamp DESC
		LIMIT ?`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	msgs, err := s.scanMessagesWithReactions(ctx, rows)
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (s *MessageStore) GetMessage(ctx context.Context, chatJID, id string) (*StoredMessage, error) {
	query := selectMessagesSQL + ` WHERE m.chat_jid = ? AND m.id = ?`
	rows, err := s.db.QueryContext(ctx, query, chatJID, id)
	if err != nil {
		return nil, err
	}
	msgs, err := s.scanMessagesWithReactions(ctx, rows)
	if err != nil || len(msgs) == 0 {
		return nil, err
	}
	return &msgs[0], nil
}

func (s *MessageStore) GetMessageByID(ctx context.Context, id string) (*StoredMessage, error) {
	query := selectMessagesSQL + ` WHERE m.id = ? LIMIT 1`
	rows, err := s.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	msgs, err := s.scanMessagesWithReactions(ctx, rows)
	if err != nil || len(msgs) == 0 {
		return nil, err
	}
	return &msgs[0], nil
}
