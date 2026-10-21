package store

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
)

// SaveHistorySync stores one history sync chunk using per-conversation transactions
// to avoid holding the database lock and blocking concurrent whatsmeow operations.
func (s *MessageStore) SaveHistorySync(ctx context.Context, client *whatsmeow.Client, data *waHistorySync.HistorySync) (int, error) {
	if pushnames := data.GetPushnames(); len(pushnames) > 0 {
		if tx, err := s.db.BeginTx(ctx, nil); err == nil {
			for _, pn := range pushnames {
				if pn.GetID() != "" && pn.GetPushname() != "" {
					_, _ = tx.ExecContext(ctx, `
						INSERT INTO whatsmeow_contacts (our_jid, their_jid, push_name)
						VALUES ('', ?, ?)
						ON CONFLICT (our_jid, their_jid) DO UPDATE SET
							push_name = CASE WHEN push_name = '' OR push_name IS NULL THEN excluded.push_name ELSE push_name END`,
						pn.GetID(), pn.GetPushname())
				}
			}
			_ = tx.Commit()
		}
	}

	stored := 0
	for _, conv := range data.GetConversations() {
		chatJID, err := types.ParseJID(conv.GetID())
		if err != nil {
			continue
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			continue
		}
		if err := s.saveConversationChat(ctx, tx, chatJID, conv); err != nil {
			_ = tx.Rollback()
			continue
		}
		n, err := s.saveConversationMessages(ctx, tx, client, chatJID, conv.GetMessages())
		if err != nil {
			_ = tx.Rollback()
			continue
		}
		if err := tx.Commit(); err == nil {
			stored += n
		}
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
