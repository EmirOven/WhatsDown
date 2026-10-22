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

	var msgID, senderJID string
	initStatus := store.MessageStatusSent
	if c.store != nil && c.store.IsDemo() {
		msgID = "demo_msg_" + time.Now().Format("20060102150405.000000")
		senderJID = store.DemoSelfJID
		initStatus = store.MessageStatusRead
	} else {
		resp, err := c.client.SendMessage(ctx, targetJID, &waMsg)
		if err != nil {
			log.Printf("sending message to %s: %v", targetJID, err)
			return
		}
		msgID = resp.ID
		senderJID = c.client.Store.ID.ToNonAD().String()
		if c.isSelfJID(chatJIDStr) {
			initStatus = store.MessageStatusRead
		}
	}

	rawProto, _ := proto.Marshal(&waMsg)
	if c.store != nil {
		sentMsg := store.StoredMessage{
			ChatJID:   chatJIDStr,
			ID:        msgID,
			SenderJID: senderJID,
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
