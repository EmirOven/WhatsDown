package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

func (s *MessageStore) GetChatOrContact(ctx context.Context, jid string) (Chat, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT jid, name, last_message_at, unread_count, archived, pinned, muted,
		       parent_jid, is_parent, is_default_sub_group
		FROM whatsdown_chats WHERE jid = ?`, jid)

	var (
		c           Chat
		lastMsgUnix int64
		archInt     int
		pinInt      int
		muteInt     int
		isParentInt int
		isDefSubInt int
	)
	err := row.Scan(
		&c.JID, &c.Name, &lastMsgUnix, &c.UnreadCount, &archInt, &pinInt, &muteInt,
		&c.ParentJID, &isParentInt, &isDefSubInt,
	)
	if err == nil {
		c.LastMessageAt = time.Unix(lastMsgUnix, 0)
		c.Archived = archInt != 0
		c.Pinned = pinInt != 0
		c.Muted = muteInt != 0
		c.IsParent = isParentInt != 0
		c.IsDefaultSubGroup = isDefSubInt != 0
		return c, nil
	}
	if err != sql.ErrNoRows {
		return c, err
	}

	c.JID = jid
	targetJID := jid
	if strings.Contains(jid, "@lid") {
		if pn := s.ResolveLIDtoPN(ctx, jid); pn != "" {
			targetJID = pn
		}
	}

	contactRow := s.db.QueryRowContext(ctx, `
		SELECT first_name, full_name, push_name, business_name
		FROM whatsmeow_contacts
		WHERE their_jid = ? OR their_jid LIKE ?
		LIMIT 1
	`, targetJID, strings.Split(targetJID, "@")[0]+"@%")

	var fn, fln, pn, bn string
	if err := contactRow.Scan(&fn, &fln, &pn, &bn); err == nil {
		name := fln
		if name == "" {
			name = fn
		}
		if name == "" {
			name = bn
		}
		if name == "" {
			name = pn
		}
		c.Name = name
	}
	return c, nil
}
