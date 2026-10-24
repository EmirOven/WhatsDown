package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/avatar"
)

func (c *MainController) newMessageDialog(title, cancelLabel string) (*gtk.Window, *gtk.Box, *adw.HeaderBar) {
	win := gtk.NewWindow()
	win.SetTitle(title)
	win.SetTransientFor(&c.window.Window)
	win.SetDestroyWithParent(true)
	win.SetModal(true)
	win.SetDefaultSize(520, 600)
	header := adw.NewHeaderBar()
	header.SetTitleWidget(adw.NewWindowTitle(title, ""))
	if cancelLabel != "" {
		header.SetShowStartTitleButtons(false)
		header.SetShowEndTitleButtons(false)
		cancel := gtk.NewButtonWithLabel(cancelLabel)
		cancel.ConnectClicked(win.Close)
		header.PackStart(cancel)
	}
	win.SetTitlebar(header)
	box := gtk.NewBox(gtk.OrientationVertical, 16)
	box.SetMarginTop(16)
	box.SetMarginBottom(16)
	box.SetMarginStart(16)
	box.SetMarginEnd(16)
	win.SetChild(box)
	keys := gtk.NewEventControllerKey()
	keys.ConnectKeyPressed(func(key, _ uint, _ gdk.ModifierType) bool {
		if key == gdk.KEY_Escape {
			win.Close()
			return true
		}
		return false
	})
	win.AddController(keys)
	return win, box, header
}

// Identity rows share sidebar avatars and names without conversation metadata.
func (c *MainController) messageIdentityRow(jid, name, subtitle string) *adw.ActionRow {
	row := adw.NewActionRow()
	row.SetUseMarkup(false)
	row.SetTitle(name)
	row.SetTitleLines(1)
	row.SetSubtitle(subtitle)
	row.SetSelectable(false)
	av := adw.NewAvatar(40, name, true)
	av.SetVAlign(gtk.AlignCenter)
	avatar.LoadAvatar(c.client, jid, name, av)
	row.AddPrefix(av)
	return row
}
