package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/avatar"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) createCommunityHeaderRow(comm store.Chat, childCount int, isExpanded bool) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.AddCSSClass("chat-row")

	box := gtk.NewBox(gtk.OrientationHorizontal, 12)
	box.SetHExpand(true)

	av := adw.NewAvatar(40, comm.Name, true)
	av.AddCSSClass("community-avatar")
	av.SetVAlign(gtk.AlignCenter)
	avatar.LoadAvatar(c.client, comm.JID, comm.Name, av, true)
	box.Append(av)

	titleBox := gtk.NewBox(gtk.OrientationVertical, 4)
	titleBox.SetHExpand(true)
	titleBox.SetVAlign(gtk.AlignCenter)

	nameLbl := gtk.NewLabel(comm.Name)
	nameLbl.SetXAlign(0)
	nameLbl.AddCSSClass("chat-row-title")
	nameLbl.SetEllipsize(3)
	titleBox.Append(nameLbl)

	subLbl := gtk.NewLabel(fmt.Sprintf("%d groups", childCount))
	subLbl.SetXAlign(0)
	subLbl.AddCSSClass("chat-row-subtitle")
	titleBox.Append(subLbl)
	box.Append(titleBox)

	arrowIcon := "pan-down-symbolic"
	if !isExpanded {
		arrowIcon = "pan-end-symbolic"
	}
	arrow := gtk.NewImageFromIconName(arrowIcon)
	arrow.AddCSSClass("dim-label")
	arrow.SetVAlign(gtk.AlignCenter)
	box.Append(arrow)

	c.attachContextMenu(row, comm, false)
	row.SetChild(box)
	return row
}

func (c *MainController) createCommunityChildRow(chat store.Chat, isArchivedView bool) *gtk.ListBoxRow {
	row := c.createChatRow(chat, isArchivedView)
	row.AddCSSClass("community-child-row")
	return row
}
