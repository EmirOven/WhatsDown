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

func (c *MainController) dispatchVoiceSend(chatJIDStr string, data []byte, seconds uint32, waveform []byte, ctxInfo *waE2E.ContextInfo) {
	ctx := context.Background()
	var msgID string
	var rawProto []byte

	if c.client != nil {
		targetJID, err := types.ParseJID(chatJIDStr)
		if err != nil {
			log.Printf("parsing JID for voice send: %v", err)
			return
		}
		uploadResp, err := c.client.Upload(ctx, data, whatsmeow.MediaAudio)
		if err != nil {
			log.Printf("uploading voice message to WhatsApp: %v", err)
			return
		}
		waMsg := buildVoiceProto(&uploadResp, uint64(len(data)), seconds, waveform, ctxInfo)
		resp, err := c.client.SendMessage(ctx, targetJID, waMsg)
		if err != nil {
			log.Printf("sending voice message: %v", err)
			return
		}
		msgID = resp.ID
		rawProto, _ = proto.Marshal(waMsg)
	} else {
		msgID = fmt.Sprintf("dev_voice_%d", time.Now().UnixNano())
		waMsg := &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				Mimetype:    proto.String("audio/ogg; codecs=opus"),
				Seconds:     &seconds,
				PTT:         proto.Bool(true),
				Waveform:    waveform,
				ContextInfo: ctxInfo,
			},
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
		Kind:        store.MessageKindVoice,
		Text:        "",
		Raw:         rawProto,
		Status:      initStatus,
		SenderSaved: true,
	}

	destPath := media.GetLocalMediaPath(sentMsg)
	_ = os.WriteFile(destPath, data, 0o600)

	if c.store != nil {
		_ = c.store.SaveMessage(ctx, sentMsg)
		_ = c.store.UpdateChatLastMessage(ctx, sentMsg.ChatJID, sentMsg.Timestamp.Unix())
		glib.IdleAdd(func() {
			c.appendSentMessageToUI(chatJIDStr, sentMsg)
		})
	}
}
