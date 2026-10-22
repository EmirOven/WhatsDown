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

	// Only the count is used here. Selecting raw would retain every sticker
	// message payload in a slice during each manual sync.
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM whatsdown_messages
  WHERE kind = 'sticker' AND raw IS NOT NULL`).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}
