package ui

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) attachParticipantContextMenu(
	row *adw.ActionRow, groupJID types.JID, p types.GroupParticipant, canManage bool, reload func(),
) {
	popover := gtk.NewPopover()
	popover.SetParent(row)
	popover.SetHasArrow(false)
	popover.AddCSSClass("popover-menu-box")

	menuBox := gtk.NewBox(gtk.OrientationVertical, 2)
	menuBox.SetMarginTop(4)
	menuBox.SetMarginBottom(4)
	menuBox.SetMarginStart(4)
	menuBox.SetMarginEnd(4)

	adminLabel := "Make Group Admin"
	adminAction := whatsmeow.ParticipantChangePromote
	if p.IsAdmin || p.IsSuperAdmin {
		adminLabel = "Dismiss as Admin"
		adminAction = whatsmeow.ParticipantChangeDemote
	}
	adminBtn := newMenuButton("security-high-symbolic", adminLabel, func() {
		if !canManage {
			return
		}
		popover.Popdown()
		go func() {
			_, _ = c.client.UpdateGroupParticipants(context.Background(), groupJID, []types.JID{p.JID}, adminAction)
			glib.IdleAdd(reload)
		}()
	})
	adminBtn.SetSensitive(canManage)
	menuBox.Append(adminBtn)

	remBtn := newMenuButton("list-remove-symbolic", "Remove from Group", func() {
		if !canManage {
			return
		}
		popover.Popdown()
		go func() {
			_, _ = c.client.UpdateGroupParticipants(context.Background(), groupJID, []types.JID{p.JID}, whatsmeow.ParticipantChangeRemove)
			glib.IdleAdd(reload)
		}()
	})
	remBtn.AddCSSClass("destructive-action")
	remBtn.SetSensitive(canManage)
	menuBox.Append(remBtn)

	popover.SetChild(menuBox)
	gesture := gtk.NewGestureClick()
	gesture.SetButton(gdk.BUTTON_SECONDARY)
	gesture.ConnectPressed(func(_ int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	row.AddController(gesture)
}

func (c *MainController) leaveAndDeleteGroup(chat store.Chat) {
	go func() {
		if c.client != nil && c.client.IsConnected() {
			if jid, err := types.ParseJID(chat.JID); err == nil {
				_ = c.client.LeaveGroup(context.Background(), jid)
			}
		}
		c.deleteChatLocally(chat)
	}()
}
