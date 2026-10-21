package ui

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) createMessageBubble(msg store.StoredMessage, sameMinuteAsPrev bool) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.SetSelectable(false)
	row.SetActivatable(false)
	row.AddCSSClass("message-row")
	if sameMinuteAsPrev {
		row.AddCSSClass("message-row-same-minute")
		row.SetMarginTop(2)
	} else {
		row.SetMarginTop(8)
	}
	row.SetMarginBottom(0)

	box := gtk.NewBox(gtk.OrientationVertical, 4)
	box.AddCSSClass("chat-bubble")

	isOutgoing := msg.FromMe || c.isSelfJID(msg.ChatJID)
	if isOutgoing {
		box.AddCSSClass("chat-bubble-outgoing")
		box.SetHAlign(gtk.AlignEnd)
	} else {
		box.AddCSSClass("chat-bubble-incoming")
		box.SetHAlign(gtk.AlignStart)
		isGroup := strings.HasSuffix(msg.ChatJID, "@g.us") || (c.currentChat != nil && strings.HasSuffix(c.currentChat.JID, "@g.us"))
		if isGroup {
			box.Append(c.buildSenderHeader(msg))
		}
	}

	if quoteWidget := c.createQuoteWidget(msg); quoteWidget != nil {
		box.Append(quoteWidget)
	}

	c.appendMessageContentWidget(box, msg)

	if textLabel := c.buildBubbleText(msg); textLabel != nil {
		box.Append(textLabel)
	}

	box.Append(c.buildBubbleFooter(msg, isOutgoing))
	c.attachMessageContextMenu(box, msg)

	rowBox := c.buildBubbleRowBox(row, box, msg, isOutgoing)
	row.SetChild(rowBox)
	return row
}

func (c *MainController) appendMessageContentWidget(box *gtk.Box, msg store.StoredMessage) {
	switch msg.Kind {
	case store.MessageKindImage, store.MessageKindVideo:
		box.AddCSSClass("chat-bubble-media")
		box.Append(c.buildImageVideoWidget(msg))
	case store.MessageKindAudio, store.MessageKindVoice:
		box.Append(c.buildAudioWidget(msg))
	case store.MessageKindDocument:
		box.Append(c.buildDocWidget(msg))
	case store.MessageKindLocation:
		box.Append(c.buildLocationWidget(msg))
	case store.MessageKindPoll:
		box.Append(c.buildPollWidget(msg))
	case store.MessageKindSticker:
		box.AddCSSClass("chat-bubble-sticker")
		box.Append(c.buildStickerWidget(msg))
	}
}
