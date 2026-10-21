package ui

import (
	"fmt"
	"strings"

	"postbetasoftware.com/whatsdown/internal/store"
)

// startReply initiates replying to a specific message.
func (c *MainController) startReply(msg store.StoredMessage) {
	c.replyingTo = &msg

	sender := "You"
	if !msg.FromMe && !c.isSelfJID(msg.ChatJID) {
		sender = msg.SenderName
		if sender == "" || strings.Contains(sender, "@") || strings.HasPrefix(sender, "+") {
			if msg.SenderJID != "" {
				if resolved := c.resolveContactName(msg.SenderJID); resolved != "" {
					sender = resolved
				}
			}
		}
		if sender == "" {
			sender = cleanPhoneNumber(msg.SenderJID)
		}
		if sender == "" {
			sender = "Sender"
		}
	}
	if c.replyPreviewSender != nil {
		c.replyPreviewSender.SetText(fmt.Sprintf("Replying to %s", sender))
	}

	previewText := truncatePreviewLines(msg.Text, 3)
	if previewText == "" {
		switch msg.Kind {
		case store.MessageKindImage:
			previewText = "📷 Photo"
		case store.MessageKindVideo:
			previewText = "🎥 Video"
		case store.MessageKindAudio, store.MessageKindVoice:
			previewText = "🎵 Audio"
		case store.MessageKindDocument:
			previewText = "📄 Document"
		case store.MessageKindLocation:
			previewText = "📍 Location"
		case store.MessageKindPoll:
			previewText = "📊 Poll"
		case store.MessageKindSticker:
			previewText = "🏷️ Sticker"
		default:
			previewText = "Message"
		}
	}
	if c.replyPreviewText != nil {
		c.replyPreviewText.SetText(previewText)
	}
	if c.replyPreviewBar != nil {
		c.replyPreviewBar.SetVisible(true)
	}
	c.scrollToBottomAnimated()
	if c.messageEntry != nil {
		c.messageEntry.GrabFocus()
	}
}

// cancelReply clears the active reply state and hides the preview bar.
func (c *MainController) cancelReply() {
	c.replyingTo = nil
	if c.replyPreviewBar != nil {
		c.replyPreviewBar.SetVisible(false)
	}
	c.scrollToBottomAnimated()
}
