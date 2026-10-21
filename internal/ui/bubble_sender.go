package ui

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildSenderHeader(msg store.StoredMessage) *gtk.Label {
	sender := msg.SenderName
	if sender == "" || strings.Contains(sender, "@") || strings.HasPrefix(sender, "+") || sender == "WhatsApp User" {
		if msg.SenderJID != "" {
			resolved := c.resolveContactName(msg.SenderJID)
			if resolved != "" && !strings.Contains(resolved, "@") && !strings.HasPrefix(resolved, "+") && resolved != "WhatsApp User" {
				sender = resolved
			} else if sender == "" || strings.Contains(sender, "@") {
				if resolved != "" && !strings.Contains(resolved, "@") {
					sender = resolved
				} else {
					cleanPhone := cleanPhoneNumber(msg.SenderJID)
					if cleanPhone != "" && !strings.Contains(cleanPhone, "@") {
						sender = cleanPhone
					}
				}
			}
		}
	}
	if sender == "" {
		sender = "Member"
	}

	if !msg.SenderSaved && sender != "" && sender != "Member" && !strings.HasPrefix(sender, "~") {
		sender = "~" + sender
	}

	colorSeed := msg.SenderJID
	if colorSeed == "" {
		colorSeed = sender
	}
	color := getSenderColor(colorSeed)

	senderLabel := gtk.NewLabel("")
	senderLabel.SetMarkup(fmt.Sprintf(`<span foreground="%s" weight="bold">%s</span>`, color, glib.MarkupEscapeText(sender)))
	senderLabel.SetXAlign(0)
	senderLabel.AddCSSClass("chat-bubble-sender")

	if msg.SenderJID != "" {
		sJID := msg.SenderJID
		gesture := gtk.NewGestureClick()
		gesture.ConnectPressed(func(nPress int, x, y float64) {
			c.openUserDetails(sJID)
		})
		senderLabel.AddController(gesture)
	}

	return senderLabel
}
