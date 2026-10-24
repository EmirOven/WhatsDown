package store

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/types"
)

func (s *MessageStore) SetChatPinned(ctx context.Context, jid string, pinned bool) error {
	v := 0
	if pinned {
		v = 1
	}
	targetJID := jid
	if alt := s.ResolveLIDtoPN(ctx, jid); alt != "" {
		targetJID = alt
	}
	altJID := s.ResolvePNtoLID(ctx, targetJID)
	res, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET pinned = ? WHERE jid = ? OR (jid = ? AND ? <> '')`, v, targetJID, altJID, altJID)
	if err == nil && pinned {
		if rows, _ := res.RowsAffected(); rows == 0 {
			_, _ = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO whatsdown_chats (jid, name, last_message_at, unread_count, pinned) VALUES (?, '', 0, 0, 1)`, targetJID)
		}
	}
	return err
}

func (s *MessageStore) SetChatArchived(ctx context.Context, jid string, archived bool) error {
	v := 0
	if archived {
		v = 1
	}
	altJID := s.ResolveLIDtoPN(ctx, jid)
	if altJID == "" {
		altJID = s.ResolvePNtoLID(ctx, jid)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET archived = ? WHERE jid = ? OR (jid = ? AND ? <> '')`, v, jid, altJID, altJID)
	return err
}

func (s *MessageStore) SetChatMuted(ctx context.Context, jid string, muted bool) error {
	v := 0
	if muted {
		v = 1
	}
	altJID := s.ResolveLIDtoPN(ctx, jid)
	if altJID == "" {
		altJID = s.ResolvePNtoLID(ctx, jid)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE whatsdown_chats SET muted = ? WHERE jid = ? OR (jid = ? AND ? <> '')`, v, jid, altJID, altJID)
	return err
}

func (s *MessageStore) MarkChatRead(ctx context.Context, chat types.JID) error {
	return s.MarkChatReadAt(ctx, chat, time.Now())
}

// MarkChatReadAt records the time of the read, so messages delivered after the
// receipt cannot bring back unread badges for messages already read on a phone.
func (s *MessageStore) MarkChatReadAt(ctx context.Context, chat types.JID, at time.Time) error {
	if at.IsZero() {
		at = time.Now()
	}
	target := chat.ToNonAD().String()
	jids := []string{target}
	if chat.Server == types.HiddenUserServer {
		if pn := s.ResolveLIDtoPN(ctx, target); pn != "" {
			jids = append(jids, pn)
		}
	} else if chat.Server == types.DefaultUserServer {
		if lid := s.ResolvePNtoLID(ctx, target); lid != "" {
			jids = append(jids, lid)
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, jid := range jids {
		if _, err := tx.ExecContext(ctx, `INSERT INTO whatsdown_chat_reads (chat_jid, read_at) VALUES (?, ?)
   ON CONFLICT(chat_jid) DO UPDATE SET read_at = MAX(read_at, excluded.read_at)`, jid, at.Unix()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE whatsdown_chats SET unread_count = (
   SELECT COUNT(*) FROM whatsdown_messages WHERE chat_jid = ? AND from_me = 0
   AND timestamp > (SELECT read_at FROM whatsdown_chat_reads WHERE chat_jid = ?)) WHERE jid = ?`, jid, jid, jid); err != nil {
			return err
		}
	}
	return tx.Commit()
}
