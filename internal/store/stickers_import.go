package store

import (
	"context"

	"go.mau.fi/whatsmeow"
)

func (s *MessageStore) ImportStickersFromChatHistory(ctx context.Context, client *whatsmeow.Client) int {
	if s.db == nil {
		return 0
	}
	s.stickerSyncMu.Lock()
	defer s.stickerSyncMu.Unlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, chat_jid, raw
		FROM whatsdown_messages
		WHERE kind = 'sticker' AND raw IS NOT NULL
		ORDER BY from_me DESC, timestamp DESC`)
	if err != nil {
		return 0
	}
	defer rows.Close()

	type item struct {
		id, chatJID string
		raw         []byte
	}
	var msgs []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.chatJID, &it.raw); err == nil {
			msgs = append(msgs, it)
		}
	}
	return len(msgs)
}
