package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

// attachMessageContextMenu adds a right-click popover context menu to a message bubble.
func (c *MainController) attachMessageContextMenu(widget gtk.Widgetter, msg store.StoredMessage) {
	w := gtk.BaseWidget(widget)
	popover := gtk.NewPopover()
	popover.SetParent(w)
	popover.SetHasArrow(false)
	popover.AddCSSClass("popover-menu-box")

	menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
	menuBox.SetMarginTop(4)
	menuBox.SetMarginBottom(4)
	menuBox.SetMarginStart(4)
	menuBox.SetMarginEnd(4)

	mCopy := msg

	if mCopy.Kind != store.MessageKindDeleted {
		menuBox.Append(c.createReactionsBar(mCopy, popover, widget))
		sep := gtk.NewSeparator(gtk.OrientationHorizontal)
		sep.SetMarginTop(2)
		sep.SetMarginBottom(4)
		menuBox.Append(sep)
	}

	if mCopy.Kind == store.MessageKindSticker {
		c.appendStickerContextActions(menuBox, popover, mCopy)
	}

	if mCopy.Kind != store.MessageKindDeleted {
		replyBtn := gtk.NewButton()
		addCSSClasses(replyBtn, "flat", "popover-menu-btn")
		rBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
		rBox.Append(gtk.NewImageFromIconName("mail-reply-sender-symbolic"))
		rBox.Append(gtk.NewLabel("Reply"))
		replyBtn.SetChild(rBox)
		replyBtn.ConnectClicked(func() {
			popover.Popdown()
			c.startReply(mCopy)
		})
		menuBox.Append(replyBtn)
	}

	if mCopy.FromMe && mCopy.Kind == store.MessageKindText {
		c.appendEditContextActions(menuBox, popover, mCopy)
	} else if mCopy.Edited {
		c.appendEditHistoryAction(menuBox, popover, mCopy)
	}

	c.appendStandardContextActions(menuBox, popover, mCopy)

	popover.SetChild(menuBox)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(3)
	gesture.ConnectPressed(func(nPress int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	w.AddController(gesture)
}
