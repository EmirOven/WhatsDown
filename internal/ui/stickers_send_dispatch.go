package ui

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) dispatchStickerSend(chatJIDStr string, data []byte, ctxInfo *waE2E.ContextInfo) {
	ctx := context.Background()
	var msgID string
	var rawProto []byte

	if c.client != nil {
		targetJID, err := types.ParseJID(chatJIDStr)
		if err != nil {
			log.Printf("parsing JID for sticker send: %v", err)
			return
		}
		uploadResp, err := c.client.Upload(ctx, data, whatsmeow.MediaImage)
		if err != nil {
			log.Printf("uploading sticker to WhatsApp: %v", err)
			return
		}
		fileLength := uint64(len(data))
		waMsg := waE2E.Message{
			StickerMessage: &waE2E.StickerMessage{
				URL:           &uploadResp.URL,
				DirectPath:    &uploadResp.DirectPath,
				MediaKey:      uploadResp.MediaKey,
				FileEncSHA256: uploadResp.FileEncSHA256,
				FileSHA256:    uploadResp.FileSHA256,
				FileLength:    &fileLength,
				Mimetype:      proto.String("image/webp"),
				Width:         proto.Uint32(512),
				Height:        proto.Uint32(512),
				ContextInfo:   ctxInfo,
			},
		}
		resp, err := c.client.SendMessage(ctx, targetJID, &waMsg)
		if err != nil {
			log.Printf("sending sticker message: %v", err)
			return
		}
		msgID = resp.ID
		rawProto, _ = proto.Marshal(&waMsg)
	} else {
		msgID = fmt.Sprintf("dev_sticker_%d", time.Now().UnixNano())
		waMsg := waE2E.Message{
			StickerMessage: &waE2E.StickerMessage{
				Mimetype:    proto.String("image/webp"),
				Width:       proto.Uint32(512),
				Height:      proto.Uint32(512),
				ContextInfo: ctxInfo,
			},
		}
		rawProto, _ = proto.Marshal(&waMsg)
	}

	initStatus := store.MessageStatusSent
	if c.isSelfJID(chatJIDStr) {
		initStatus = store.MessageStatusRead
	}
	sentMsg := store.StoredMessage{
		ChatJID:     chatJIDStr,
		ID:          msgID,
		SenderJID:   "me",
		FromMe:      true,
		Timestamp:   time.Now(),
		Kind:        store.MessageKindSticker,
		Text:        "",
		Raw:         rawProto,
		Status:      initStatus,
		SenderSaved: true,
	}

	if c.store != nil {
		_ = c.store.SaveMessage(ctx, sentMsg)
		_ = c.store.UpdateChatLastMessage(ctx, sentMsg.ChatJID, sentMsg.Timestamp.Unix())
		glib.IdleAdd(func() {
			c.appendSentMessageToUI(chatJIDStr, sentMsg)
		})
	}
}
