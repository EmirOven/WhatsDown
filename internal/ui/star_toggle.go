package ui

import (
	"context"
	"log"
	"postbetasoftware.com/whatsdown/internal/wasync"
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/store"
)

// toggleMessageStarred flips the starred status of a message.
func (c *MainController) toggleMessageStarred(msg store.StoredMessage) {
	if c.store == nil {
		return
	}
	newStarred := !msg.Starred
	go func() {
		ctx := context.Background()
		_ = c.store.ToggleMessageStarred(ctx, msg.ChatJID, msg.ID, newStarred)

		chatJIDStr := msg.ChatJID
		if strings.HasSuffix(chatJIDStr, "@lid") && c.store != nil {
			chatJIDStr = c.store.ResolveLIDtoPN(ctx, chatJIDStr)
		}

		if c.client != nil && c.client.IsConnected() {
			chatJID, err1 := types.ParseJID(chatJIDStr)
			if err1 == nil {
				var senderJID types.JID
				if strings.HasSuffix(chatJIDStr, "@g.us") {
					senderJIDStr := msg.SenderJID
					if strings.HasSuffix(senderJIDStr, "@lid") && c.store != nil {
						senderJIDStr = c.store.ResolveLIDtoPN(ctx, senderJIDStr)
					}
					senderJID, _ = types.ParseJID(senderJIDStr)
				} else {
					senderJID = chatJID
				}
				if err := wasync.SendAppState(ctx, c.client, appstate.BuildStar(chatJID, senderJID, types.MessageID(msg.ID), msg.FromMe, newStarred)); err != nil {
					log.Printf("SendAppState BuildStar failed for %s (%s): %v", chatJIDStr, msg.ID, err)
				}
			}
		}

		msgID := msg.ID
		glib.IdleAdd(func() {
			if c.currentChat != nil && c.isSameChat(c.currentChat.JID, msg.ChatJID) {
				c.updateSingleMessageRow(msgID)
			}
		})
	}()
}
