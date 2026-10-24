package ui

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) dispatchMediaSend(
	chatJIDStr, filePath, filename, caption, kind, mimeType string,
	appMediaType whatsmeow.MediaType,
	data []byte,
	ctxInfo *waE2E.ContextInfo,
) {
	ctx := context.Background()
	var msgID string
	var rawProto []byte

	if c.client != nil {
		targetJID, err := types.ParseJID(chatJIDStr)
		if err != nil {
			log.Printf("parsing JID for media send: %v", err)
			return
		}
		uploadResp, err := c.client.Upload(ctx, data, appMediaType)
		if err != nil {
			log.Printf("uploading media to WhatsApp: %v", err)
			return
		}
		waMsg := buildMediaProto(kind, caption, filename, mimeType, &uploadResp, uint64(len(data)), ctxInfo)
		resp, err := c.client.SendMessage(ctx, targetJID, waMsg)
		if err != nil {
			log.Printf("sending media message: %v", err)
			return
		}
		msgID = resp.ID
		rawProto, _ = proto.Marshal(waMsg)
	} else {
		msgID = fmt.Sprintf("dev_media_%d", time.Now().UnixNano())
		waMsg := &waE2E.Message{
			Conversation: proto.String(caption),
		}
		rawProto, _ = proto.Marshal(waMsg)
	}

	initStatus := store.MessageStatusSent
	if c.isSelfJID(chatJIDStr) {
		initStatus = store.MessageStatusRead
	}
	sentMsg := store.StoredMessage{
		ChatJID:     chatJIDStr,
		ID:          msgID,
		SenderName:  "You",
		FromMe:      true,
		Timestamp:   time.Now(),
		Kind:        kind,
		Text:        caption,
		Raw:         rawProto,
		Status:      initStatus,
		SenderSaved: true,
	}

	destPath := media.GetLocalMediaPath(sentMsg)
	_ = os.WriteFile(destPath, data, 0o600)

	if c.store != nil {
		_ = c.saveOutgoingMessage(ctx, sentMsg)
		_ = c.store.UpdateChatLastMessage(ctx, sentMsg.ChatJID, sentMsg.Timestamp.Unix())
		glib.IdleAdd(func() {
			c.appendSentMessageToUI(chatJIDStr, sentMsg)
		})
	}
}
