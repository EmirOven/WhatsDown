package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildBubbleRowBox(row *gtk.ListBoxRow, box *gtk.Box, msg store.StoredMessage, isOutgoing bool) *gtk.Box {
	rowBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	rowBox.AddCSSClass("message-row-box")

	bubbleContainer := gtk.NewBox(gtk.OrientationVertical, 0)
	bubbleContainer.Append(box)

	if len(msg.Reactions) > 0 {
		if reactionPill := c.buildReactionPill(msg); reactionPill != nil {
			reactionWrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
			reactionWrapper.AddCSSClass("reaction-pill-wrapper")
			if isOutgoing {
				reactionWrapper.SetHAlign(gtk.AlignEnd)
				reactionWrapper.SetMarginEnd(18)
			} else {
				reactionWrapper.SetHAlign(gtk.AlignStart)
				reactionWrapper.SetMarginStart(18)
			}
			reactionWrapper.Append(reactionPill)
			bubbleContainer.Append(reactionWrapper)
		}
	}

	if msg.Kind != store.MessageKindDeleted {
		mCopy := msg
		replyBtn := gtk.NewButton()
		addCSSClasses(replyBtn, "flat", "circular", "message-reply-btn")
		replyBtn.SetIconName("mail-reply-sender-symbolic")
		replyBtn.SetTooltipText("Reply")
		replyBtn.SetVAlign(gtk.AlignCenter)
		replyBtn.SetCanFocus(false)
		replyBtn.SetOpacity(0.0)

		replyBtn.ConnectClicked(func() {
			if replyBtn.Opacity() < 0.2 {
				return
			}
			c.startReply(mCopy)
		})

		motion := gtk.NewEventControllerMotion()
		motion.ConnectEnter(func(x, y float64) { replyBtn.SetOpacity(1.0) })
		motion.ConnectLeave(func() { replyBtn.SetOpacity(0.0) })
		row.AddController(motion)

		if isOutgoing {
			rowBox.SetHAlign(gtk.AlignEnd)
			rowBox.Append(replyBtn)
			rowBox.Append(bubbleContainer)
		} else {
			rowBox.SetHAlign(gtk.AlignStart)
			rowBox.Append(bubbleContainer)
			rowBox.Append(replyBtn)
		}
	} else {
		if isOutgoing {
			rowBox.SetHAlign(gtk.AlignEnd)
		} else {
			rowBox.SetHAlign(gtk.AlignStart)
		}
		rowBox.Append(bubbleContainer)
	}

	return rowBox
}
