package store

import (
	"context"
)

// DeleteChatLocally permanently removes all local records and messages for a chat from SQLite.
func (s *MessageStore) DeleteChatLocally(ctx context.Context, jid string) error {
	jids := []string{jid}
	if alt := s.ResolveLIDtoPN(ctx, jid); alt != "" {
		jids = append(jids, alt)
	}
	if alt := s.ResolvePNtoLID(ctx, jid); alt != "" {
		jids = append(jids, alt)
	}
	if s.IsSelfJID(ctx, jid) {
		pn, lid := s.GetSelfJIDs(ctx)
		if pn != "" {
			jids = append(jids, pn)
		}
		if lid != "" {
			jids = append(jids, lid)
		}
	}

	for _, j := range jids {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_messages WHERE chat_jid = ?`, j)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_message_edits WHERE chat_jid = ?`, j)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_reactions WHERE chat_jid = ?`, j)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_poll_votes WHERE chat_jid = ?`, j)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_calls WHERE chat_jid = ?`, j)
		_, _ = s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET parent_jid = '' WHERE parent_jid = ?`, j)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_chats WHERE jid = ?`, j)
	}
	return nil
}
