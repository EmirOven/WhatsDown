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

	// Menu buttons capture the popover, which owns the buttons natively.
	// Disconnect those callbacks before removing the row to break that cycle.
	var disconnect []func()
	menuButton := func(icon, label string, action func()) *gtk.Button {
		btn := newMenuButtonContent(icon, label)
		handle := btn.ConnectClicked(action)
		disconnect = append(disconnect, func() { btn.HandlerDisconnect(handle) })
		return btn
	}

	menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
	menuBox.SetMarginTop(4)
	menuBox.SetMarginBottom(4)
	menuBox.SetMarginStart(4)
	menuBox.SetMarginEnd(4)

	pinText := "Pin chat"
	if chat.Pinned {
		pinText = "Unpin chat"
	}
	menuBox.Append(menuButton("view-pin-symbolic", pinText, func() {
		popover.Popdown()
		c.togglePin(chat)
	}))

	muteText, muteIcon := "Mute chat", "audio-volume-muted-symbolic"
	if chat.Muted {
		muteText, muteIcon = "Unmute chat", "audio-volume-high-symbolic"
	}
	menuBox.Append(menuButton(muteIcon, muteText, func() {
		popover.Popdown()
		c.toggleMute(chat)
	}))

	menuBox.Append(menuButton("help-about-symbolic", "Details & Media", func() {
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
	menuBox.Append(menuButton(archIcon, archText, func() {
		popover.Popdown()
		c.toggleArchive(chat)
	}))

	sep2 := gtk.NewSeparator(gtk.OrientationHorizontal)
	sep2.SetMarginTop(2)
	sep2.SetMarginBottom(2)
	menuBox.Append(sep2)

	delBtn := menuButton("user-trash-symbolic", "Delete chat", func() {
		popover.Popdown()
		c.deleteChatLocally(chat)
	})
	delBtn.AddCSSClass("destructive-action")
	menuBox.Append(delBtn)

	popover.SetChild(menuBox)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(gdk.BUTTON_SECONDARY)
	gestureHandle := gesture.ConnectPressed(func(nPress int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	row.AddController(gesture)
	if c.chatMenuCleanups == nil {
		c.chatMenuCleanups = make(map[*gtk.ListBoxRow]func())
	}
	c.chatMenuCleanups[row] = func() {
		gesture.HandlerDisconnect(gestureHandle)
		for _, f := range disconnect {
			f()
		}
		popover.Popdown()
		popover.Unparent()
	}
}

func newMenuButton(iconName, label string, onClick func()) *gtk.Button {
	btn := newMenuButtonContent(iconName, label)
	btn.ConnectClicked(onClick)
	return btn
}

func newMenuButtonContent(iconName, label string) *gtk.Button {
	btn := gtk.NewButton()
	btn.AddCSSClass("flat")
	btn.AddCSSClass("popover-menu-btn")
	b := gtk.NewBox(gtk.OrientationHorizontal, 10)
	b.Append(gtk.NewImageFromIconName(iconName))
	b.Append(gtk.NewLabel(label))
	btn.SetChild(b)
	return btn
}
