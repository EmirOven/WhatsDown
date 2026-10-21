package ui

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) dispatchTextMessage(targetJID types.JID, chatJIDStr, text string, ctxInfo *waE2E.ContextInfo) {
	ctx := context.Background()
	var waMsg waE2E.Message
	if ctxInfo != nil {
		waMsg = waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text:        proto.String(text),
				ContextInfo: ctxInfo,
			},
		}
	} else {
		waMsg = waE2E.Message{
			Conversation: proto.String(text),
		}
	}

	resp, err := c.client.SendMessage(ctx, targetJID, &waMsg)
	if err != nil {
		log.Printf("sending message to %s: %v", targetJID, err)
		return
	}

	rawProto, _ := proto.Marshal(&waMsg)
	initStatus := store.MessageStatusSent
	if c.isSelfJID(chatJIDStr) {
		initStatus = store.MessageStatusRead
	}
	if c.store != nil {
		sentMsg := store.StoredMessage{
			ChatJID:   chatJIDStr,
			ID:        resp.ID,
			SenderJID: c.client.Store.ID.ToNonAD().String(),
			FromMe:    true,
			Timestamp: time.Now(),
			Kind:      store.MessageKindText,
			Text:      text,
			Raw:       rawProto,
			Status:    initStatus,
		}
		_ = c.store.SaveMessage(ctx, sentMsg)
		_ = c.store.UpdateChatLastMessage(ctx, sentMsg.ChatJID, sentMsg.Timestamp.Unix())
		glib.IdleAdd(func() {
			c.appendSentMessageToUI(chatJIDStr, sentMsg)
		})
	}
}
