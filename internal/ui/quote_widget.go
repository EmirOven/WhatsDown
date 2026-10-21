package ui

import (
	"context"
	"fmt"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) createQuoteWidget(msg store.StoredMessage) *gtk.Box {
	ctxInfo := media.ExtractContextInfo(msg.Raw)
	if ctxInfo == nil {
		return nil
	}
	stanzaID := ctxInfo.GetStanzaID()
	participant := ctxInfo.GetParticipant()
	quotedRaw := ctxInfo.GetQuotedMessage()
	if stanzaID == "" && quotedRaw == nil {
		return nil
	}

	var origMsg *store.StoredMessage
	if c.store != nil && stanzaID != "" {
		origMsg, _ = c.store.GetMessageByID(context.Background(), stanzaID)
	}

	quotedSenderName := c.resolveQuotedSender(origMsg, participant, msg.FromMe)
	quotedSnippet := c.resolveQuotedSnippet(origMsg, quotedRaw)

	quoteBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	quoteBox.AddCSSClass("chat-bubble-quote")
	quoteBox.SetCursorFromName("pointer")

	contentBox := gtk.NewBox(gtk.OrientationVertical, 2)
	contentBox.SetHExpand(true)

	colorSeed := participant
	if colorSeed == "" {
		colorSeed = quotedSenderName
	}
	authorColor := getSenderColor(colorSeed)
	if quotedSenderName == "You" {
		authorColor = "#3584e4"
	}

	authorLabel := gtk.NewLabel("")
	authorLabel.SetMarkup(fmt.Sprintf(`<span foreground="%s" weight="bold">%s</span>`, authorColor, glib.MarkupEscapeText(quotedSenderName)))
	authorLabel.SetXAlign(0)
	authorLabel.AddCSSClass("chat-bubble-quote-sender")
	contentBox.Append(authorLabel)

	textLabel := gtk.NewLabel(quotedSnippet)
	textLabel.SetXAlign(0)
	textLabel.SetWrap(true)
	textLabel.SetWrapMode(pango.WrapWordChar)
	textLabel.SetLines(3)
	textLabel.SetEllipsize(pango.EllipsizeEnd)
	textLabel.AddCSSClass("chat-bubble-quote-text")
	contentBox.Append(textLabel)

	quoteBox.Append(contentBox)

	if thumbPic := getQuoteThumbnail(origMsg, quotedRaw); thumbPic != nil {
		quoteBox.Append(thumbPic)
	}

	if stanzaID != "" {
		targetID := stanzaID
		gesture := gtk.NewGestureClick()
		gesture.ConnectPressed(func(nPress int, x, y float64) {
			c.scrollToMessageID(targetID)
		})
		quoteBox.AddController(gesture)
	}

	return quoteBox
}
