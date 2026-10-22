package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/avatar"
)

func (c *MainController) createMentionRow(item GroupMemberItem, idx int) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.AddCSSClass("mention-row")

	box := gtk.NewBox(gtk.OrientationHorizontal, 8)
	box.SetMarginStart(8)
	box.SetMarginEnd(8)
	box.SetMarginTop(4)
	box.SetMarginBottom(4)

	av := adw.NewAvatar(32, item.DisplayName, true)
	if item.JID != "all" {
		avatar.LoadAvatar(c.client, item.JID, item.DisplayName, av)
	}
	box.Append(av)

	textBox := gtk.NewBox(gtk.OrientationVertical, 2)
	textBox.SetHExpand(true)

	dispTitle := item.DisplayName
	if item.JID == "all" {
		dispTitle = "@all"
	}
	nameLbl := gtk.NewLabel(dispTitle)
	nameLbl.SetXAlign(0)
	nameLbl.AddCSSClass("mention-row-name")
	nameLbl.SetEllipsize(pango.EllipsizeEnd)
	textBox.Append(nameLbl)

	if item.Subtitle != "" && item.Subtitle != item.DisplayName {
		subLbl := gtk.NewLabel(item.Subtitle)
		subLbl.SetXAlign(0)
		subLbl.AddCSSClass("dim-label")
		subLbl.AddCSSClass("caption")
		subLbl.SetEllipsize(pango.EllipsizeEnd)
		textBox.Append(subLbl)
	}
	box.Append(textBox)

	row.SetChild(box)
	return row
}
