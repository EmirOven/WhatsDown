package store

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
)

func (s *MessageStore) saveConversationChat(ctx context.Context, tx execer, chatJID types.JID, conv *waHistorySync.Conversation) error {
	targetJIDStr := chatJID.String()
	if strings.HasSuffix(targetJIDStr, "@lid") {
		if pn := s.ResolveLIDtoPN(ctx, targetJIDStr); pn != "" {
			targetJIDStr = pn
		}
	}
	name := conv.GetDisplayName()
	if name == "" {
		name = conv.GetName()
	}
	if name == "" {
		_ = tx.QueryRowContext(ctx, `
			SELECT COALESCE(
				NULLIF(cnt.full_name, ''),
				NULLIF(cnt_lid.full_name, ''),
				NULLIF(cnt.first_name, ''),
				NULLIF(cnt_lid.first_name, ''),
				NULLIF(cnt.business_name, ''),
				NULLIF(cnt_lid.business_name, ''),
				NULLIF(cnt.push_name, ''),
				NULLIF(cnt_lid.push_name, ''),
				''
			)
			FROM (SELECT ? AS jid) t
			LEFT JOIN whatsmeow_contacts cnt ON cnt.their_jid = t.jid
			LEFT JOIN whatsmeow_lid_map lm ON (lm.lid || '@lid') = t.jid
			LEFT JOIN whatsmeow_contacts cnt_lid ON cnt_lid.their_jid = (lm.pn || '@s.whatsapp.net')`,
			targetJIDStr).Scan(&name)
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO whatsdown_chats (jid, name, last_message_at, unread_count, archived, pinned)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (jid) DO UPDATE SET
			name            = CASE WHEN excluded.name <> '' THEN excluded.name ELSE name END,
			last_message_at = MAX(last_message_at, excluded.last_message_at),
			unread_count    = excluded.unread_count,
			archived        = CASE WHEN excluded.archived = 1 THEN 1 ELSE archived END,
			pinned          = CASE WHEN excluded.pinned = 1 THEN 1 ELSE pinned END`,
		targetJIDStr, name, int64(conv.GetConversationTimestamp()),
		conv.GetUnreadCount(), conv.GetArchived(), conv.GetPinned() > 0)
	return err
}

func (s *MessageStore) saveConversationMessages(ctx context.Context, tx execer, client *whatsmeow.Client, chatJID types.JID, messages []*waHistorySync.HistorySyncMsg) (int, error) {
	stored := 0
	for _, hsm := range messages {
		evt, err := client.ParseWebMessage(chatJID, hsm.GetMessage())
		if err != nil {
			continue
		}
		isStarred := hsm.GetMessage().GetStarred()
		status := MessageStatusNone
		if evt.Info.IsFromMe {
			switch hsm.GetMessage().GetStatus() {
			case waWeb.WebMessageInfo_SERVER_ACK:
				status = MessageStatusSent
			case waWeb.WebMessageInfo_DELIVERY_ACK:
				status = MessageStatusDelivered
			case waWeb.WebMessageInfo_READ, waWeb.WebMessageInfo_PLAYED:
				status = MessageStatusRead
			default:
				status = MessageStatusSent
			}
		}
		inserted, err := s.SaveMessageTxWithStar(ctx, tx, evt, isStarred, status)
		if err != nil {
			return 0, err
		}
		if inserted {
			stored++
		}
	}
	return stored, nil
}
