package ui

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) dispatchPollSend(chatJIDStr, question string, options []string, selectableCount int, ctxInfo *waE2E.ContextInfo) {
	ctx := context.Background()
	var msgID string
	var rawProto []byte

	if c.client != nil && (c.store == nil || !c.store.IsDemo()) {
		targetJID, err := types.ParseJID(chatJIDStr)
		if err != nil {
			log.Printf("parsing JID for poll send: %v", err)
			return
		}
		waMsg := c.client.BuildPollCreation(question, options, selectableCount)
		if ctxInfo != nil && waMsg != nil && waMsg.PollCreationMessage != nil {
			waMsg.PollCreationMessage.ContextInfo = ctxInfo
		}
		resp, err := c.client.SendMessage(ctx, targetJID, waMsg)
		if err != nil {
			log.Printf("sending poll message: %v", err)
			return
		}
		msgID = resp.ID
		rawProto, _ = proto.Marshal(waMsg)
	} else {
		msgID = fmt.Sprintf("dev_poll_%d", time.Now().UnixNano())
		var optProtos []*waE2E.PollCreationMessage_Option
		for _, opt := range options {
			optProtos = append(optProtos, &waE2E.PollCreationMessage_Option{OptionName: proto.String(opt)})
		}
		waMsg := &waE2E.Message{
			PollCreationMessage: &waE2E.PollCreationMessage{
				Name:                   proto.String(question),
				Options:                optProtos,
				SelectableOptionsCount: proto.Uint32(uint32(selectableCount)),
				ContextInfo:            ctxInfo,
			},
		}
		rawProto, _ = proto.Marshal(waMsg)
	}

	var ownJID string
	if c.client != nil && c.client.Store != nil && c.client.Store.ID != nil {
		ownJID = c.client.Store.ID.ToNonAD().String()
	}

	initStatus := store.MessageStatusSent
	if c.isSelfJID(chatJIDStr) {
		initStatus = store.MessageStatusRead
	}
	sentMsg := store.StoredMessage{
		ChatJID:     chatJIDStr,
		ID:          msgID,
		SenderJID:   ownJID,
		SenderName:  "You",
		FromMe:      true,
		Timestamp:   time.Now(),
		Kind:        store.MessageKindPoll,
		Text:        question,
		Raw:         rawProto,
		Status:      initStatus,
		SenderSaved: true,
	}

	if c.store != nil {
		_ = c.saveOutgoingMessage(ctx, sentMsg)
		_ = c.store.UpdateChatLastMessage(ctx, sentMsg.ChatJID, sentMsg.Timestamp.Unix())
		glib.IdleAdd(func() {
			c.appendSentMessageToUI(chatJIDStr, sentMsg)
		})
	}
}
