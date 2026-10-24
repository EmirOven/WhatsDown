package app

import (
	"context"
	"log"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
)

func handleReceiptEvent(e *events.Receipt) {
	if msgStore == nil {
		return
	}
	// Our other devices reading a chat is separate from recipients reading our message.
	if e.Type == types.ReceiptTypeReadSelf || e.Type == types.ReceiptTypePlayedSelf {
		go func() {
			if err := msgStore.MarkChatReadAt(context.Background(), e.Chat, e.Timestamp); err != nil {
				log.Printf("marking chat read: %v", err)
				return
			}
			refreshChatsUI()
		}()
		return
	}
	status := store.MessageStatusNone
	switch e.Type {
	case types.ReceiptTypeRead, types.ReceiptTypePlayed:
		status = store.MessageStatusRead
	case types.ReceiptTypeDelivered, types.ReceiptTypeInactive:
		status = store.MessageStatusDelivered
	}
	// Sender receipts acknowledge delivery to our own devices, not to recipients.
	if status == store.MessageStatusNone || len(e.MessageIDs) == 0 || e.IsFromMe {
		return
	}
	msgIDs := make([]string, len(e.MessageIDs))
	for i, id := range e.MessageIDs {
		msgIDs[i] = string(id)
	}
	chat := e.Chat.ToNonAD().String()
	if e.Chat.Server == types.HiddenUserServer {
		if canonical := msgStore.ResolveLIDtoPN(context.Background(), chat); canonical != "" {
			chat = canonical
		}
	}
	sender := e.Sender.ToNonAD()
	if e.SenderAlt.Server == types.DefaultUserServer {
		sender = e.SenderAlt.ToNonAD()
	}
	go func() {
		if err := msgStore.SaveMessageReceipt(context.Background(), chat, msgIDs, sender.String(), status, e.Timestamp); err != nil {
			log.Printf("saving message receipt: %v", err)
			return
		}
		queueReceiptRefresh(chat, msgIDs)
	}()
}
