package ui

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/emoji"
	"postbetasoftware.com/whatsdown/internal/markdown"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func hasQuotedMessage(msg store.StoredMessage) bool {
	if len(msg.Raw) == 0 {
		return false
	}
	ctxInfo := media.ExtractContextInfo(msg.Raw)
	if ctxInfo == nil {
		return false
	}
	return ctxInfo.GetStanzaID() != "" || ctxInfo.GetQuotedMessage() != nil
}

func (c *MainController) isSingleEmojiMessage(msg store.StoredMessage) bool {
	if msg.Kind != "" && msg.Kind != store.MessageKindText {
		return false
	}
	if hasQuotedMessage(msg) {
		return false
	}
	return emoji.IsSingleEmoji(msg.Text)
}

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

		if c.isSingleEmojiMessage(msg) {
			textLabel.AddCSSClass("chat-bubble-single-emoji")
			textLabel.SetXAlign(0.5)
			markup = fmt.Sprintf(`<span font="64pt">%s</span>`, markup)
		}

		textLabel.SetUseMarkup(true)
		textLabel.SetMarkup(markup)

		if hasMentions {
			c.trackWidgetSignal(textLabel, textLabel, textLabel.ConnectActivateLink(func(uri string) bool {
				if uri == "user:all" {
					return true
				}
				if strings.HasPrefix(uri, "user:") {
					targetJID := strings.TrimPrefix(uri, "user:")
					c.openUserDetails(targetJID)
					return true
				}
				return false
			}))
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
