package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildBubbleFooter(msg store.StoredMessage, isOutgoing bool) *gtk.Box {
	footerBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	footerBox.SetHAlign(gtk.AlignEnd)
	footerBox.SetVAlign(gtk.AlignCenter)
	footerBox.AddCSSClass("chat-bubble-footer")
	if msg.Kind == store.MessageKindSticker {
		footerBox.AddCSSClass("chat-bubble-sticker-footer")
	}

	if msg.Starred {
		starIcon := gtk.NewImageFromIconName("starred-symbolic")
		starIcon.AddCSSClass("pin-indicator")
		starIcon.SetVAlign(gtk.AlignCenter)
		footerBox.Append(starIcon)
	}

	timeStr := msg.Timestamp.Local().Format("15:04")
	if msg.Edited {
		timeStr += " (edited)"
	}
	timeLabel := gtk.NewLabel(timeStr)
	timeLabel.SetXAlign(1)
	timeLabel.SetVAlign(gtk.AlignCenter)
	timeLabel.AddCSSClass("chat-bubble-time")
	footerBox.Append(timeLabel)

	if isOutgoing {
		status := msg.Status
		if c.isSelfJID(msg.ChatJID) {
			status = store.MessageStatusRead
		}
		statusLabel := gtk.NewLabel("")
		statusLabel.SetVAlign(gtk.AlignCenter)
		statusLabel.AddCSSClass("chat-bubble-status")
		c.applyStatusToLabel(statusLabel, status)
		footerBox.Append(statusLabel)
		if c.messageStatusLabels != nil {
			c.messageStatusLabels[msg.ID] = statusLabel
		}
	}

	return footerBox
}

// applyStatusToLabel sets checkmark text and CSS classes based on message status.
func (c *MainController) applyStatusToLabel(lbl *gtk.Label, status int) {
	if lbl == nil {
		return
	}
	lbl.RemoveCSSClass("chat-bubble-status-sent")
	lbl.RemoveCSSClass("chat-bubble-status-delivered")
	lbl.RemoveCSSClass("chat-bubble-status-read")

	switch status {
	case store.MessageStatusRead:
		lbl.SetMarkup(`<span letter_spacing="-2048">✓✓</span>`)
		lbl.AddCSSClass("chat-bubble-status-read")
	case store.MessageStatusDelivered:
		lbl.SetMarkup(`<span letter_spacing="-2048">✓✓</span>`)
		lbl.AddCSSClass("chat-bubble-status-delivered")
	case store.MessageStatusSent:
		lbl.SetText("✓")
		lbl.AddCSSClass("chat-bubble-status-sent")
	default:
		lbl.SetText("✓")
		lbl.AddCSSClass("chat-bubble-status-sent")
	}
}
