package ui

import (
	"context"
	"sort"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) appendGroupManagementSections(dialog *adw.Dialog, chat store.Chat, contentBox *gtk.Box) {
	settingsGroup := adw.NewPreferencesGroup()
	settingsGroup.SetTitle("Group Settings & Actions")

	adminRow := adw.NewActionRow()
	adminRow.SetTitle("Admin-only messages")
	adminRow.SetSubtitle("Only admins can send messages in this group")
	adminSwitch := gtk.NewSwitch()
	adminSwitch.SetVAlign(gtk.AlignCenter)
	adminSwitch.SetSensitive(false)
	adminRow.AddSuffix(adminSwitch)
	adminRow.SetActivatableWidget(adminSwitch)
	settingsGroup.Add(adminRow)

	addRow := adw.NewActionRow()
	addRow.SetTitle("Add Participants")
	addRow.SetSubtitle("Invite contacts to this group")
	addRow.AddPrefix(gtk.NewImageFromIconName("contact-new-symbolic"))
	addRow.SetActivatable(true)
	settingsGroup.Add(addRow)

	delRow := adw.NewActionRow()
	delRow.SetTitle("Delete Group Chat")
	delRow.SetSubtitle("Leave group and remove from chat list")
	delRow.AddPrefix(gtk.NewImageFromIconName("user-trash-symbolic"))
	delRow.AddCSSClass("error")
	delRow.SetActivatable(true)
	delRow.ConnectActivated(func() {
		dialog.Close()
		c.leaveAndDeleteGroup(chat)
	})
	settingsGroup.Add(delRow)

	membersGroup := adw.NewPreferencesGroup()
	membersGroup.SetTitle("Participants")
	membersList := gtk.NewListBox()
	membersList.SetSelectionMode(gtk.SelectionNone)
	membersList.AddCSSClass("boxed-list")
	membersGroup.Add(membersList)

	contentBox.Append(settingsGroup)
	contentBox.Append(membersGroup)

	var currentMembers []types.GroupParticipant
	var reload func()
	reload = func() {
		c.loadGroupDetailsState(chat.JID, adminSwitch, membersGroup, membersList, &currentMembers, reload)
	}
	addRow.ConnectActivated(func() {
		c.showAddGroupMembersDialog(chat.JID, currentMembers, reload)
	})
	reload()
}

func (c *MainController) loadGroupDetailsState(
	groupJID string,
	adminSwitch *gtk.Switch,
	membersGroup *adw.PreferencesGroup,
	membersList *gtk.ListBox,
	currentMembers *[]types.GroupParticipant,
	reload func(),
) {
	if c.client == nil || !c.client.IsConnected() {
		return
	}
	go func() {
		jid, err := types.ParseJID(groupJID)
		if err != nil {
			return
		}
		info, err := c.client.GetGroupInfo(context.Background(), jid)
		if err != nil || info == nil {
			return
		}
		parts := append([]types.GroupParticipant(nil), info.Participants...)
		sort.SliceStable(parts, func(i, j int) bool {
			return parts[i].IsAdmin && !parts[j].IsAdmin
		})
		glib.IdleAdd(func() {
			*currentMembers = parts
			c.populateGroupMembersList(jid, info, adminSwitch, membersGroup, membersList, parts, reload)
		})
	}()
}
