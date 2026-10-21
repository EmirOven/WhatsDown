package store

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

func (s *MessageStore) handleProtocolMessage(ctx context.Context, tx execer, chatJID string, p *waE2E.ProtocolMessage) (bool, error) {
	targetID := p.GetKey().GetID()
	if targetID == "" {
		return false, nil
	}

	switch p.GetType() {
	case waE2E.ProtocolMessage_REVOKE:
		_, err := tx.ExecContext(ctx, `
			UPDATE whatsdown_messages
			SET kind = ?, text = '', raw = NULL
			WHERE chat_jid = ? AND id = ?`,
			MessageKindDeleted, chatJID, targetID)
		return false, err

	case waE2E.ProtocolMessage_MESSAGE_EDIT:
		edited := p.GetEditedMessage()
		if edited == nil {
			return false, nil
		}
		kind, text, ok := DescribeMessage(edited)
		if !ok {
			return false, nil
		}
		now := time.Now().Unix()
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO whatsdown_message_edits (message_id, chat_jid, text, edited_at)
			VALUES (?, ?, ?, ?)`,
			targetID, chatJID, text, now)

		_, err := tx.ExecContext(ctx, `
			UPDATE whatsdown_messages
			SET kind = ?, text = ?, edited = 1
			WHERE chat_jid = ? AND id = ?`,
			kind, text, chatJID, targetID)
		return false, err
	}
	return false, nil
}

func (s *MessageStore) insertOrUpdateMessage(ctx context.Context, tx execer, chatJID, msgID, senderJID, pushName string, fromMe bool, ts int64, kind, text string, raw []byte, starInt, st int, incrementUnread bool) (bool, error) {
	fromMeInt := 0
	if fromMe {
		fromMeInt = 1
	}

	_, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO whatsdown_chats (jid, name, last_message_at, unread_count)
		VALUES (?, ?, ?, 0)`,
		chatJID, pushName, ts)
	if err != nil {
		return false, err
	}

	res, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO whatsdown_messages
			(chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, starred, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chatJID, msgID, senderJID, pushName, fromMeInt, ts, kind, text, raw, starInt, st)
	if err != nil {
		return false, err
	}

	rows, _ := res.RowsAffected()
	if rows > 0 && !fromMe && incrementUnread {
		_, _ = tx.ExecContext(ctx, `
			UPDATE whatsdown_chats
			SET unread_count = unread_count + 1,
			    last_message_at = MAX(last_message_at, ?)
			WHERE jid = ?`,
			ts, chatJID)
	} else if rows > 0 {
		_, _ = tx.ExecContext(ctx, `
			UPDATE whatsdown_chats
			SET last_message_at = MAX(last_message_at, ?)
			WHERE jid = ?`,
			ts, chatJID)
	}
	return rows > 0, nil
}
