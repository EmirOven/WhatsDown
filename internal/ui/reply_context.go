package ui

import (
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

// buildReplyContextInfo constructs a waE2E.ContextInfo quoting the specified message.
func (c *MainController) buildReplyContextInfo(quoted store.StoredMessage) *waE2E.ContextInfo {
	var quotedRaw *waE2E.Message
	if len(quoted.Raw) > 0 {
		quotedRaw = media.ParseE2EMessage(quoted.Raw)
	}
	if quotedRaw == nil {
		quotedRaw = &waE2E.Message{
			Conversation: proto.String(quoted.Text),
		}
	} else {
		quotedRaw = proto.Clone(quotedRaw).(*waE2E.Message)
		if quotedRaw.ExtendedTextMessage != nil {
			quotedRaw.ExtendedTextMessage.ContextInfo = nil
		}
		if quotedRaw.ImageMessage != nil {
			quotedRaw.ImageMessage.ContextInfo = nil
		}
		if quotedRaw.VideoMessage != nil {
			quotedRaw.VideoMessage.ContextInfo = nil
		}
		if quotedRaw.AudioMessage != nil {
			quotedRaw.AudioMessage.ContextInfo = nil
		}
		if quotedRaw.DocumentMessage != nil {
			quotedRaw.DocumentMessage.ContextInfo = nil
		}
		if quotedRaw.StickerMessage != nil {
			quotedRaw.StickerMessage.ContextInfo = nil
		}
	}

	var participant string
	if quoted.FromMe || c.isSelfJID(quoted.ChatJID) {
		if c.client != nil && c.client.Store != nil && c.client.Store.ID != nil {
			participant = c.client.Store.ID.ToNonAD().String()
		}
	} else {
		participant = quoted.SenderJID
		if participant == "" {
			participant = quoted.ChatJID
		}
	}
	if pj, err := types.ParseJID(participant); err == nil {
		participant = pj.ToNonAD().String()
	}

	ctxInfo := &waE2E.ContextInfo{
		StanzaID:      proto.String(quoted.ID),
		QuotedMessage: quotedRaw,
	}
	if participant != "" {
		ctxInfo.Participant = proto.String(participant)
	}
	return ctxInfo
}
