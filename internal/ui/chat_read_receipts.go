package ui

import (
	"context"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
)

func (c *MainController) sendReadReceipts(targetJID types.JID, chatJIDStr string) {
	ctx := context.Background()
	targetAppJID := targetJID
	if strings.HasSuffix(targetAppJID.String(), "@lid") && c.store != nil {
		if pnJID, err := types.ParseJID(c.store.ResolveLIDtoPN(ctx, targetAppJID.String())); err == nil {
			targetAppJID = pnJID
		}
	}
	_ = c.client.SendAppState(ctx, appstate.BuildMarkChatAsRead(targetAppJID, true, time.Now(), nil))

	if c.store == nil {
		return
	}
	msgs, err := c.store.ListMessages(ctx, chatJIDStr, time.Time{}, 50)
	if err != nil {
		return
	}
	senderMap := make(map[types.JID][]types.MessageID)
	now := time.Now()
	for _, m := range msgs {
		if m.FromMe {
			continue
		}
		sJID, err := types.ParseJID(m.SenderJID)
		if err != nil {
			sJID = targetJID
		}
		senderMap[sJID] = append(senderMap[sJID], types.MessageID(m.ID))
	}
	for sJID, ids := range senderMap {
		if len(ids) > 0 {
			_ = c.client.MarkRead(ctx, ids, now, targetJID, sJID)
		}
	}
}
