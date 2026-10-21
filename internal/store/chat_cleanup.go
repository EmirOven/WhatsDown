package store

import (
	"context"
	"strings"
)

func (s *MessageStore) ResolvePNtoLID(ctx context.Context, jid string) string {
	pnPart := jid
	if at := strings.Index(pnPart, "@"); at != -1 {
		pnPart = pnPart[:at]
	}
	var lid string
	_ = s.db.QueryRowContext(ctx, `SELECT lid FROM whatsmeow_lid_map WHERE pn = ?`, pnPart).Scan(&lid)
	if lid != "" {
		return lid + "@lid"
	}
	return ""
}

func (s *MessageStore) CleanupDuplicateChats(ctx context.Context) {
	s.cleanupDuplicateChats(ctx)
}

func (s *MessageStore) cleanupDuplicateChats(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT lm.pn || '@s.whatsapp.net', lm.lid || '@lid'
		FROM whatsmeow_lid_map lm
		JOIN whatsdown_chats c_lid ON c_lid.jid = (lm.lid || '@lid')
		JOIN whatsdown_chats c_pn ON c_pn.jid = (lm.pn || '@s.whatsapp.net')
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	var pairs [][2]string
	for rows.Next() {
		var pnJID, lidJID string
		if err := rows.Scan(&pnJID, &lidJID); err == nil && pnJID != "" && lidJID != "" {
			pairs = append(pairs, [2]string{pnJID, lidJID})
		}
	}
	_ = rows.Close()

	for _, pair := range pairs {
		pnJID, lidJID := pair[0], pair[1]
		_, _ = s.db.ExecContext(ctx, `
			UPDATE whatsdown_chats
			SET last_message_at = MAX(last_message_at, (SELECT last_message_at FROM whatsdown_chats WHERE jid = ?)),
			    unread_count    = unread_count + (SELECT unread_count FROM whatsdown_chats WHERE jid = ?),
			    pinned          = MAX(pinned, (SELECT pinned FROM whatsdown_chats WHERE jid = ?)),
			    muted           = MAX(muted, (SELECT muted FROM whatsdown_chats WHERE jid = ?)),
			    archived        = MAX(archived, (SELECT archived FROM whatsdown_chats WHERE jid = ?)),
			    name            = CASE WHEN name = '' THEN (SELECT name FROM whatsdown_chats WHERE jid = ?) ELSE name END
			WHERE jid = ?`,
			lidJID, lidJID, lidJID, lidJID, lidJID, lidJID, pnJID)
		_, _ = s.db.ExecContext(ctx, `UPDATE OR IGNORE whatsdown_messages SET chat_jid = ? WHERE chat_jid = ?`, pnJID, lidJID)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_messages WHERE chat_jid = ?`, lidJID)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM whatsdown_chats WHERE jid = ?`, lidJID)
	}
}
