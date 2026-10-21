package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) attachReplyMotion(row *gtk.ListBoxRow, box *gtk.Box, replyBtn *gtk.Button, msg store.StoredMessage) {
	mCopy := msg
	doubleClick := gtk.NewGestureClick()
	doubleClick.SetButton(1)
	doubleClick.SetPropagationPhase(gtk.PhaseCapture)
	doubleClick.ConnectPressed(func(nPress int, x, y float64) {
		if nPress == 2 {
			c.startReply(mCopy)
		}
	})
	box.AddController(doubleClick)

	replyBtn.ConnectClicked(func() {
		if replyBtn.Opacity() < 0.2 {
			return
		}
		c.startReply(mCopy)
	})

	motion := gtk.NewEventControllerMotion()
	motion.ConnectEnter(func(x, y float64) {
		replyBtn.SetOpacity(1.0)
		c.adjustReplyButtonPosition(row, replyBtn)
	})
	motion.ConnectMotion(func(x, y float64) {
		c.adjustReplyButtonPosition(row, replyBtn)
	})
	motion.ConnectLeave(func() {
		replyBtn.SetOpacity(0.0)
	})
	row.AddController(motion)
}

func (c *MainController) adjustReplyButtonPosition(row *gtk.ListBoxRow, replyBtn *gtk.Button) {
	rowHeight := row.Height()
	if rowHeight <= 80 || c.messagesScrolledWindow == nil {
		replyBtn.SetVAlign(gtk.AlignCenter)
		replyBtn.SetMarginTop(0)
		return
	}

	bounds, ok := row.ComputeBounds(c.messagesScrolledWindow)
	if !ok || bounds == nil {
		replyBtn.SetVAlign(gtk.AlignCenter)
		replyBtn.SetMarginTop(0)
		return
	}

	viewHeight := float32(c.messagesScrolledWindow.Height())
	if viewHeight <= 0 {
		return
	}

	rowY := bounds.Y()
	visTop := -rowY
	if visTop < 0 {
		visTop = 0
	}
	visBottom := viewHeight - rowY
	if visBottom > float32(rowHeight) {
		visBottom = float32(rowHeight)
	}

	if visBottom <= visTop {
		return
	}

	targetY := (visTop + visBottom) / 2
	btnMargin := int(targetY) - 14
	if btnMargin < 4 {
		btnMargin = 4
	}
	if btnMargin > rowHeight-32 {
		btnMargin = rowHeight - 32
	}

	replyBtn.SetVAlign(gtk.AlignStart)
	replyBtn.SetMarginTop(btnMargin)
}
