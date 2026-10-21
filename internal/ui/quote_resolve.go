package ui

import (
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) resolveQuotedSender(origMsg *store.StoredMessage, participant string, msgFromMe bool) string {
	if origMsg != nil {
		if origMsg.FromMe || c.isSelfJID(origMsg.SenderJID) {
			return "You"
		}
		sender := origMsg.SenderName
		if sender == "" || strings.Contains(sender, "@") || strings.HasPrefix(sender, "+") {
			if origMsg.SenderJID != "" {
				if resolved := c.resolveContactName(origMsg.SenderJID); resolved != "" {
					sender = resolved
				}
			}
		}
		if sender == "" {
			sender = cleanPhoneNumber(origMsg.SenderJID)
		}
		if sender != "" {
			return sender
		}
	}
	if c.isSelfJID(participant) {
		return "You"
	} else if participant != "" {
		if resolved := c.resolveContactName(participant); resolved != "" {
			return resolved
		}
		return cleanPhoneNumber(participant)
	}
	if msgFromMe {
		return "You"
	}
	return "Message"
}

func (c *MainController) resolveQuotedSnippet(origMsg *store.StoredMessage, quotedRaw *waE2E.Message) string {
	if origMsg != nil && origMsg.Text != "" {
		return origMsg.Text
	}
	if quotedRaw != nil {
		snippet, _ := media.SummarizeProtoMessage(quotedRaw)
		if snippet != "" {
			return snippet
		}
	}
	if origMsg != nil {
		switch origMsg.Kind {
		case store.MessageKindImage:
			return "📷 Photo"
		case store.MessageKindVideo:
			return "🎥 Video"
		case store.MessageKindAudio, store.MessageKindVoice:
			return "🎵 Audio"
		case store.MessageKindDocument:
			return "📄 Document"
		case store.MessageKindLocation:
			return "📍 Location"
		case store.MessageKindPoll:
			return "📊 Poll"
		case store.MessageKindSticker:
			return "🏷️ Sticker"
		default:
			return "Message"
		}
	}
	return "Original message"
}
