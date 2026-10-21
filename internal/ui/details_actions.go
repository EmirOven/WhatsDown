package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) attachMediaContextMenu(widget gtk.Widgetter, msg store.StoredMessage, dialog *adw.Dialog) {
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
	cLabel := "Copy Media"
	if mCopy.Kind == store.MessageKindText {
		cLabel = "Copy Text"
	}
	copyBtn := newMenuButton("edit-copy-symbolic", cLabel, func() {
		popover.Popdown()
		if mCopy.Kind == store.MessageKindText {
			gdk.DisplayGetDefault().Clipboard().SetText(mCopy.Text)
		} else {
			media.CopyMediaToClipboard(c.client, mCopy)
		}
	})
	menuBox.Append(copyBtn)

	if mCopy.Kind != store.MessageKindText && mCopy.Kind != store.MessageKindDeleted {
		showBtn := newMenuButton("folder-open-symbolic", "Show in File Manager", func() {
			popover.Popdown()
			media.ShowMediaInFileManager(c.client, mCopy)
		})
		menuBox.Append(showBtn)
	}

	goBtn := newMenuButton("go-jump-symbolic", "Go to Message", func() {
		popover.Popdown()
		if dialog != nil {
			dialog.Close()
		}
		c.jumpToMessage(mCopy.ChatJID, mCopy.ID)
	})
	menuBox.Append(goBtn)

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
