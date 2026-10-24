package store

import "context"

// DeleteMessage removes local message data, or leaves a tombstone after revocation.
func (s *MessageStore) DeleteMessage(ctx context.Context, chatJID, id string, revoked bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"whatsdown_message_edits", "whatsdown_reactions", "whatsdown_poll_votes"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE chat_jid = ? AND message_id = ?", chatJID, id); err != nil {
			return err
		}
	}
	for _, table := range []string{"whatsdown_message_recipients", "whatsdown_message_recipient_sets"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE chat_jid = ? AND message_id = ?", chatJID, id); err != nil {
			return err
		}
	}
	if revoked {
		_, err = tx.ExecContext(ctx, `UPDATE whatsdown_messages SET kind = ?, text = '', raw = NULL WHERE chat_jid = ? AND id = ?`, MessageKindDeleted, chatJID, id)
	} else {
		_, err = tx.ExecContext(ctx, `DELETE FROM whatsdown_messages WHERE chat_jid = ? AND id = ?`, chatJID, id)
	}
	if err != nil {
		return err
	}
	if !revoked {
		if _, err := tx.ExecContext(ctx, `UPDATE whatsdown_chats SET last_message_at = COALESCE((SELECT MAX(timestamp) FROM whatsdown_messages WHERE chat_jid = ?), 0) WHERE jid = ?`, chatJID, chatJID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
