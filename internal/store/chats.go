package store

import (
	"context"
	"time"
)

func (s *MessageStore) ListChats(ctx context.Context) ([]Chat, error) {
	return s.fetchChats(ctx, false)
}

func (s *MessageStore) ListArchivedChats(ctx context.Context) ([]Chat, error) {
	return s.fetchChats(ctx, true)
}

func (s *MessageStore) fetchChats(ctx context.Context, archived bool) ([]Chat, error) {
	archivedInt := 0
	if archived {
		archivedInt = 1
	}

	query := `
		SELECT 
			c.jid, c.name, c.last_message_at, c.unread_count, c.archived, c.pinned, c.muted,
			c.parent_jid, c.is_parent, c.is_default_sub_group,
			COALESCE(m.text, ''), COALESCE(m.kind, ''), COALESCE(m.from_me, 0), COALESCE(m.status, 0), COALESCE(NULLIF(m.sender_name, ''), m.sender_jid, '')
		FROM whatsdown_chats c
		LEFT JOIN (
			SELECT chat_jid, text, kind, from_me, status, sender_jid, sender_name
			FROM (
				SELECT chat_jid, text, kind, from_me, status, sender_jid, sender_name,
				       ROW_NUMBER() OVER (PARTITION BY chat_jid ORDER BY timestamp DESC, rowid DESC) as rn
				FROM whatsdown_messages
			) WHERE rn = 1
		) m ON c.jid = m.chat_jid
		WHERE c.archived = ?
		ORDER BY c.pinned DESC, c.last_message_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, archivedInt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var (
			c           Chat
			lastMsgUnix int64
			archInt     int
			pinInt      int
			muteInt     int
			isParentInt int
			isDefSubInt int
			fromMeInt   int
		)
		err := rows.Scan(
			&c.JID, &c.Name, &lastMsgUnix, &c.UnreadCount, &archInt, &pinInt, &muteInt,
			&c.ParentJID, &isParentInt, &isDefSubInt,
			&c.LastMessageText, &c.LastMessageKind, &fromMeInt, &c.LastMessageStatus, &c.LastMessageSender,
		)
		if err != nil {
			return nil, err
		}
		c.LastMessageAt = time.Unix(lastMsgUnix, 0)
		c.Archived = archInt != 0
		c.Pinned = pinInt != 0
		c.Muted = muteInt != 0
		c.IsParent = isParentInt != 0
		c.IsDefaultSubGroup = isDefSubInt != 0
		c.LastMessageFromMe = fromMeInt != 0
		chats = append(chats, c)
	}
	return chats, rows.Err()
}
