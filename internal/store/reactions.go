package store

import (
	"context"
	"time"
)

func (s *MessageStore) GetReactionsForChat(ctx context.Context, chatJID string) (map[string][]Reaction, error) {
	query := `
		SELECT chat_jid, message_id, sender_jid, sender_name, emoji, from_me, timestamp
		FROM whatsdown_reactions
		WHERE chat_jid = ?
		ORDER BY timestamp ASC`
	rows, err := s.db.QueryContext(ctx, query, chatJID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]Reaction)
	for rows.Next() {
		var r Reaction
		var fromMeInt int
		var ts int64
		if err := rows.Scan(&r.ChatJID, &r.MessageID, &r.SenderJID, &r.SenderName, &r.Emoji, &fromMeInt, &ts); err != nil {
			return nil, err
		}
		r.FromMe = fromMeInt != 0
		r.Timestamp = time.Unix(ts, 0)
		result[r.MessageID] = append(result[r.MessageID], r)
	}
	return result, rows.Err()
}

func (s *MessageStore) GetReactionsForMessage(ctx context.Context, chatJID, messageID string) ([]Reaction, error) {
	query := `
		SELECT chat_jid, message_id, sender_jid, sender_name, emoji, from_me, timestamp
		FROM whatsdown_reactions
		WHERE chat_jid = ? AND message_id = ?
		ORDER BY timestamp ASC`
	rows, err := s.db.QueryContext(ctx, query, chatJID, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Reaction
	for rows.Next() {
		var r Reaction
		var fromMeInt int
		var ts int64
		if err := rows.Scan(&r.ChatJID, &r.MessageID, &r.SenderJID, &r.SenderName, &r.Emoji, &fromMeInt, &ts); err != nil {
			return nil, err
		}
		r.FromMe = fromMeInt != 0
		r.Timestamp = time.Unix(ts, 0)
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *MessageStore) SaveReaction(ctx context.Context, r Reaction) error {
	fromMeInt := 0
	if r.FromMe {
		fromMeInt = 1
	}
	query := `
		INSERT INTO whatsdown_reactions (chat_jid, message_id, sender_jid, sender_name, emoji, from_me, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (chat_jid, message_id, sender_jid) DO UPDATE SET
			emoji       = excluded.emoji,
			sender_name = CASE WHEN excluded.sender_name <> '' THEN excluded.sender_name ELSE whatsdown_reactions.sender_name END,
			timestamp   = excluded.timestamp`
	_, err := s.db.ExecContext(ctx, query, r.ChatJID, r.MessageID, r.SenderJID, r.SenderName, r.Emoji, fromMeInt, r.Timestamp.Unix())
	return err
}

func (s *MessageStore) RemoveReaction(ctx context.Context, chatJID, messageID, senderJID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM whatsdown_reactions
		WHERE chat_jid = ? AND message_id = ? AND sender_jid = ?`,
		chatJID, messageID, senderJID)
	return err
}

func (s *MessageStore) RemoveMyReaction(ctx context.Context, chatJID, messageID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM whatsdown_reactions
		WHERE chat_jid = ? AND message_id = ? AND from_me = 1`,
		chatJID, messageID)
	return err
}
