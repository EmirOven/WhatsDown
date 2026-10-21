package store

import "context"

// SaveMessage inserts or updates a single stored message in the database.
func (s *MessageStore) SaveMessage(ctx context.Context, msg StoredMessage) error {
	fromMeInt := 0
	if msg.FromMe {
		fromMeInt = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (chat_jid, id) DO UPDATE SET status = MAX(whatsdown_messages.status, excluded.status)`,
		msg.ChatJID, msg.ID, msg.SenderJID, msg.SenderName, fromMeInt, msg.Timestamp.Unix(), msg.Kind, msg.Text, msg.Raw, msg.Status)
	return err
}

// UpdateChatLastMessage updates the last message timestamp for a chat.
func (s *MessageStore) UpdateChatLastMessage(ctx context.Context, chatJID string, lastMessageAt int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE whatsdown_chats SET last_message_at = ? WHERE jid = ?`,
		lastMessageAt, chatJID)
	return err
}
