package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) attachStickerPickerMenu(btn *gtk.Button, stk store.SavedSticker) {
	itemPopover := gtk.NewPopover()
	itemPopover.SetParent(btn)
	itemPopover.SetHasArrow(true)
	itemPopover.AddCSSClass("popover-menu-box")

	popBox := gtk.NewBox(gtk.OrientationVertical, 2)
	popBox.SetMarginTop(4)
	popBox.SetMarginBottom(4)
	popBox.SetMarginStart(4)
	popBox.SetMarginEnd(4)

	sendItem := gtk.NewButton()
	addCSSClasses(sendItem, "flat", "popover-menu-btn")
	sRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
	sRow.Append(gtk.NewImageFromIconName("mail-send-symbolic"))
	sRow.Append(gtk.NewLabel("Send Sticker"))
	sendItem.SetChild(sRow)
	sendItem.ConnectClicked(func() {
		itemPopover.Popdown()
		if c.stickerPopover != nil {
			c.stickerPopover.Popdown()
		}
		c.sendSticker(stk.FilePath)
	})
	popBox.Append(sendItem)

	delItem := gtk.NewButton()
	addCSSClasses(delItem, "flat", "popover-menu-btn")
	dRow := gtk.NewBox(gtk.OrientationHorizontal, 8)
	dRow.Append(gtk.NewImageFromIconName("user-trash-symbolic"))
	dRow.Append(gtk.NewLabel("Remove from Saved"))
	delItem.SetChild(dRow)
	delItem.ConnectClicked(func() {
		itemPopover.Popdown()
		c.removeSavedSticker(stk.ID)
	})
	popBox.Append(delItem)

	itemPopover.SetChild(popBox)

	rGesture := gtk.NewGestureClick()
	rGesture.SetButton(3)
	rGesture.ConnectPressed(func(nPress int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		itemPopover.SetPointingTo(&rect)
		itemPopover.Popup()
	})
	btn.AddController(rGesture)
}
