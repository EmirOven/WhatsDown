package ui

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/markdown"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildBubbleText(msg store.StoredMessage) *gtk.Label {
	displayText := msg.Text
	if displayText != "" && msg.Kind != store.MessageKindPoll {
		mentionedJIDs := media.ExtractMentionedJIDs(msg.Raw)
		markup, hasMentions := markdown.FormatForBubble(displayText, mentionedJIDs, c.resolveContactName)

		textLabel := gtk.NewLabel("")
		textLabel.SetWrap(true)
		textLabel.SetWrapMode(pango.WrapWordChar)
		textLabel.SetXAlign(0)
		textLabel.SetSelectable(true)
		textLabel.AddCSSClass("chat-bubble-text")

		if _, _, _, err := pango.ParseMarkup(markup, 0); err == nil {
			textLabel.SetMarkup(markup)
		} else {
			textLabel.SetText(displayText)
		}

		if hasMentions {
			textLabel.ConnectActivateLink(func(uri string) bool {
				if strings.HasPrefix(uri, "user:") {
					targetJID := strings.TrimPrefix(uri, "user:")
					c.openUserDetails(targetJID)
					return true
				}
				return false
			})
		}
		return textLabel
	}

	if msg.Kind != store.MessageKindImage && msg.Kind != store.MessageKindVideo &&
		msg.Kind != store.MessageKindAudio && msg.Kind != store.MessageKindVoice &&
		msg.Kind != store.MessageKindDocument && msg.Kind != store.MessageKindLocation &&
		msg.Kind != store.MessageKindSticker {
		switch msg.Kind {
		case store.MessageKindPoll:
			displayText = "📊 Poll"
		case store.MessageKindDeleted:
			displayText = "🚫 This message was deleted"
		default:
			displayText = "💬 Message"
		}
		textLabel := gtk.NewLabel(displayText)
		textLabel.SetWrap(true)
		textLabel.SetWrapMode(pango.WrapWordChar)
		textLabel.SetXAlign(0)
		textLabel.SetSelectable(true)
		textLabel.AddCSSClass("chat-bubble-text")
		return textLabel
	}

	return nil
}
