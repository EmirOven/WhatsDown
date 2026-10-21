package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) attachContextMenu(row *gtk.ListBoxRow, chat store.Chat, isArchivedView bool) {
	popover := gtk.NewPopover()
	popover.SetParent(row)
	popover.SetHasArrow(false)
	popover.AddCSSClass("popover-menu-box")

	menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
	menuBox.SetMarginTop(4)
	menuBox.SetMarginBottom(4)
	menuBox.SetMarginStart(4)
	menuBox.SetMarginEnd(4)

	pinText := "Pin chat"
	if chat.Pinned {
		pinText = "Unpin chat"
	}
	menuBox.Append(newMenuButton("view-pin-symbolic", pinText, func() {
		popover.Popdown()
		c.togglePin(chat)
	}))

	muteText, muteIcon := "Mute chat", "audio-volume-muted-symbolic"
	if chat.Muted {
		muteText, muteIcon = "Unmute chat", "audio-volume-high-symbolic"
	}
	menuBox.Append(newMenuButton(muteIcon, muteText, func() {
		popover.Popdown()
		c.toggleMute(chat)
	}))

	menuBox.Append(newMenuButton("help-about-symbolic", "Details & Media", func() {
		popover.Popdown()
		c.showDetailsAndMedia(chat)
	}))

	sep := gtk.NewSeparator(gtk.OrientationHorizontal)
	sep.SetMarginTop(2)
	sep.SetMarginBottom(2)
	menuBox.Append(sep)

	archText, archIcon := "Archive chat", "folder-download-symbolic"
	if chat.Archived {
		archText, archIcon = "Unarchive chat", "folder-symbolic"
	}
	menuBox.Append(newMenuButton(archIcon, archText, func() {
		popover.Popdown()
		c.toggleArchive(chat)
	}))

	sep2 := gtk.NewSeparator(gtk.OrientationHorizontal)
	sep2.SetMarginTop(2)
	sep2.SetMarginBottom(2)
	menuBox.Append(sep2)

	delBtn := newMenuButton("user-trash-symbolic", "Delete chat", func() {
		popover.Popdown()
		c.deleteChatLocally(chat)
	})
	delBtn.AddCSSClass("destructive-action")
	menuBox.Append(delBtn)

	popover.SetChild(menuBox)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(gdk.BUTTON_SECONDARY)
	gesture.ConnectPressed(func(nPress int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	row.AddController(gesture)
}

func newMenuButton(iconName, label string, onClick func()) *gtk.Button {
	btn := gtk.NewButton()
	btn.AddCSSClass("flat")
	btn.AddCSSClass("popover-menu-btn")
	b := gtk.NewBox(gtk.OrientationHorizontal, 10)
	b.Append(gtk.NewImageFromIconName(iconName))
	b.Append(gtk.NewLabel(label))
	btn.SetChild(b)
	btn.ConnectClicked(onClick)
	return btn
}
