package store

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
)

// SaveHistorySync stores one history sync chunk in a single transaction.
func (s *MessageStore) SaveHistorySync(ctx context.Context, client *whatsmeow.Client, data *waHistorySync.HistorySync) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stored := 0
	for _, pn := range data.GetPushnames() {
		if pn.GetID() != "" && pn.GetPushname() != "" {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO whatsmeow_contacts (our_jid, their_jid, push_name)
				VALUES ('', ?, ?)
				ON CONFLICT (our_jid, their_jid) DO UPDATE SET
					push_name = CASE WHEN push_name = '' OR push_name IS NULL THEN excluded.push_name ELSE push_name END`,
				pn.GetID(), pn.GetPushname())
		}
	}

	for _, conv := range data.GetConversations() {
		chatJID, err := types.ParseJID(conv.GetID())
		if err != nil {
			continue
		}
		if err := s.saveConversationChat(ctx, tx, chatJID, conv); err != nil {
			return 0, err
		}
		n, err := s.saveConversationMessages(ctx, tx, client, chatJID, conv.GetMessages())
		if err != nil {
			return 0, err
		}
		stored += n
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	favs := data.GetFavoriteStickers()
	recents := data.GetRecentStickers()
	if (len(favs) > 0 || len(recents) > 0) && client != nil {
		go s.SyncHistoryStickers(context.Background(), client, favs, recents)
	}
	if client != nil {
		go s.ImportStickersFromChatHistory(context.Background(), client)
	}

	return stored, nil
}
