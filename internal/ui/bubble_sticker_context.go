package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) appendStickerContextActions(menuBox *gtk.Box, popover *gtk.Popover, mCopy store.StoredMessage) {
	isSaved := c.isStickerSaved(mCopy)
	saveBtn := gtk.NewButton()
	addCSSClasses(saveBtn, "flat", "popover-menu-btn")
	saveRow := gtk.NewBox(gtk.OrientationHorizontal, 10)
	if isSaved {
		saveRow.Append(gtk.NewImageFromIconName("user-trash-symbolic"))
		saveRow.Append(gtk.NewLabel("Remove from Saved Stickers"))
		saveBtn.SetChild(saveRow)
		saveBtn.ConnectClicked(func() {
			popover.Popdown()
			c.removeSavedStickerByMessage(mCopy)
		})
	} else {
		saveRow.Append(gtk.NewImageFromIconName("star-new-symbolic"))
		saveRow.Append(gtk.NewLabel("Save Sticker"))
		saveBtn.SetChild(saveRow)
		saveBtn.ConnectClicked(func() {
			popover.Popdown()
			c.saveStickerFromMessage(mCopy)
		})
	}
	menuBox.Append(saveBtn)

	openBtn := gtk.NewButton()
	addCSSClasses(openBtn, "flat", "popover-menu-btn")
	openRow := gtk.NewBox(gtk.OrientationHorizontal, 10)
	openRow.Append(gtk.NewImageFromIconName("insert-image-symbolic"))
	openRow.Append(gtk.NewLabel("Open as Image"))
	openBtn.SetChild(openRow)
	openBtn.ConnectClicked(func() {
		popover.Popdown()
		media.OpenMedia(c.client, mCopy, nil)
	})
	menuBox.Append(openBtn)
}

func (c *MainController) attachStickerContextMenu(widget gtk.Widgetter, msg store.StoredMessage) {
	w := gtk.BaseWidget(widget)
	popover := gtk.NewPopover()
	popover.SetParent(w)
	popover.SetHasArrow(true)
	popover.AddCSSClass("popover-menu-box")

	mCopy := msg
	populate := func() {
		menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
		menuBox.SetMarginTop(4)
		menuBox.SetMarginBottom(4)
		menuBox.SetMarginStart(4)
		menuBox.SetMarginEnd(4)

		menuBox.Append(c.createReactionsBar(mCopy, popover, widget))
		sep := gtk.NewSeparator(gtk.OrientationHorizontal)
		sep.SetMarginTop(2)
		sep.SetMarginBottom(4)
		menuBox.Append(sep)

		c.appendStickerContextActions(menuBox, popover, mCopy)

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

		c.appendStandardContextActions(menuBox, popover, mCopy)
		popover.SetChild(menuBox)
	}

	gesture := gtk.NewGestureClick()
	gesture.SetButton(0)
	gesture.ConnectPressed(func(nPress int, x, y float64) {
		populate()
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	w.AddController(gesture)
}
