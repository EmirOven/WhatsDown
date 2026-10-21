package store

import (
	"context"
)

func (s *MessageStore) cleanupSelfChats(ctx context.Context) {
	canonical := s.GetCanonicalSelfJID(ctx)
	if canonical == "" || canonical == "me" {
		return
	}
	rows, err := s.db.QueryContext(ctx, `SELECT jid FROM whatsdown_chats WHERE jid != ?`, canonical)
	if err != nil {
		return
	}
	defer rows.Close()

	var others []string
	for rows.Next() {
		var j string
		if err := rows.Scan(&j); err == nil && s.IsSelfJID(ctx, j) {
			others = append(others, j)
		}
	}
	_ = rows.Close()

	for _, other := range others {
		_, _ = s.db.ExecContext(ctx, `UPDATE OR IGNORE whatsdown_messages SET chat_jid = ? WHERE chat_jid = ?`, canonical, other)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_messages WHERE chat_jid = ?`, other)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_chats WHERE jid = ?`, other)
	}

	var maxTime int64
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(timestamp), 0) FROM whatsdown_messages WHERE chat_jid = ?`, canonical).Scan(&maxTime)
	if maxTime > 0 {
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO whatsdown_chats (jid, name, last_message_at, unread_count)
			VALUES (?, '', ?, 0)
			ON CONFLICT (jid) DO UPDATE SET last_message_at = MAX(last_message_at, excluded.last_message_at)`,
			canonical, maxTime)
	}
}
