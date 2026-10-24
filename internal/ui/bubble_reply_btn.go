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
	c.trackWidgetSignal(box, doubleClick, doubleClick.ConnectPressed(func(nPress int, x, y float64) {
		if nPress == 2 && messageTargetAllowsReply(box.Pick(x, y, gtk.PickDefault)) {
			c.startReply(mCopy)
		}
	}))
	box.AddController(doubleClick)

	c.trackWidgetSignal(replyBtn, replyBtn, replyBtn.ConnectClicked(func() {
		if replyBtn.Opacity() < 0.2 {
			return
		}
		c.startReply(mCopy)
	}))

	motion := gtk.NewEventControllerMotion()
	c.trackWidgetSignal(row, motion, motion.ConnectEnter(func(x, y float64) {
		replyBtn.SetOpacity(1.0)
		c.adjustReplyButtonPosition(row, replyBtn)
	}))
	c.trackWidgetSignal(row, motion, motion.ConnectMotion(func(x, y float64) {
		c.adjustReplyButtonPosition(row, replyBtn)
	}))
	c.trackWidgetSignal(row, motion, motion.ConnectLeave(func() {
		replyBtn.SetOpacity(0.0)
	}))
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

// Capture-phase reply gestures must leave embedded controls to their own actions.
func messageTargetAllowsReply(target gtk.Widgetter) bool {
	for target != nil {
		w := gtk.BaseWidget(target)
		if w.HasCSSClass("message-interactive") {
			return false
		}
		switch target.(type) {
		case *gtk.Button, *gtk.Scale, *gtk.Range, *gtk.Entry, *gtk.TextView, *gtk.DropDown:
			return false
		}
		target = w.Parent()
	}
	return true
}
