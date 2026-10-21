package app

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func handleReceiptEvent(e *events.Receipt) {
	status := store.MessageStatusNone
	switch e.Type {
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf, types.ReceiptTypePlayed, types.ReceiptTypePlayedSelf:
		status = store.MessageStatusRead
	case types.ReceiptTypeDelivered, types.ReceiptTypeSender:
		status = store.MessageStatusDelivered
	}

	if e.Type == types.ReceiptTypeRead || e.Type == types.ReceiptTypeReadSelf {
		go func() {
			if msgStore != nil {
				_ = msgStore.MarkChatRead(context.Background(), e.Chat)
				glib.IdleAdd(func() {
					if active := ui.ActiveMainWindow(); active != nil {
						active.LoadChats()
						active.LoadArchivedChats()
					}
				})
			}
		}()
	}

	if status > store.MessageStatusNone && len(e.MessageIDs) > 0 {
		msgIDs := make([]string, len(e.MessageIDs))
		for i, mid := range e.MessageIDs {
			msgIDs[i] = string(mid)
		}
		go func() {
			if msgStore != nil {
				_ = msgStore.UpdateMessageStatuses(context.Background(), msgIDs, status)
			}
			glib.IdleAdd(func() {
				if active := ui.ActiveMainWindow(); active != nil {
					active.HandleReceiptUpdate(msgIDs, status)
				}
			})
		}()
	}
}
