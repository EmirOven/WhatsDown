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
	permWidgets := buildGroupPermissionsGroup()

	actionsGroup := adw.NewPreferencesGroup()
	actionsGroup.SetTitle("Group Actions")

	addRow := adw.NewActionRow()
	addRow.SetTitle("Add Participants")
	addRow.SetSubtitle("Invite contacts to this group")
	addRow.AddPrefix(gtk.NewImageFromIconName("contact-new-symbolic"))
	addRow.SetActivatable(true)
	addRow.SetSensitive(false)
	actionsGroup.Add(addRow)
	c.addGroupInviteActionRows(actionsGroup, chat.JID, permWidgets)

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
	actionsGroup.Add(delRow)

	membersGroup := adw.NewPreferencesGroup()
	membersGroup.SetTitle("Participants")
	membersList := gtk.NewListBox()
	membersList.SetSelectionMode(gtk.SelectionNone)
	membersList.AddCSSClass("boxed-list")
	membersGroup.Add(membersList)

	contentBox.Append(permWidgets.group)
	contentBox.Append(actionsGroup)
	contentBox.Append(membersGroup)

	var currentMembers []types.GroupParticipant
	var reload func()
	reload = func() {
		c.loadGroupDetailsState(chat.JID, permWidgets, addRow, membersGroup, membersList, &currentMembers, reload)
	}
	addRow.ConnectActivated(func() {
		c.showAddGroupMembersDialog(chat.JID, currentMembers, reload)
	})
	reload()
}

func (c *MainController) loadGroupDetailsState(
	groupJID string,
	permWidgets *groupPermWidgets,
	addRow *adw.ActionRow,
	membersGroup *adw.PreferencesGroup,
	membersList *gtk.ListBox,
	currentMembers *[]types.GroupParticipant,
	reload func(),
) {
	isDemo := c.store != nil && c.store.IsDemo()
	if !isDemo && (c.client == nil || !c.client.IsConnected()) {
		return
	}
	go func() {
		jid, err := types.ParseJID(groupJID)
		if err != nil {
			return
		}
		var ext *groupExtendedInfo
		if isDemo {
			ext = c.buildDemoGroupExtendedInfo(jid)
		} else {
			ext, err = c.fetchGroupExtendedInfo(context.Background(), jid)
		}
		if err != nil || ext == nil || ext.Info == nil {
			return
		}
		parts := append([]types.GroupParticipant(nil), ext.Info.Participants...)
		sort.SliceStable(parts, func(i, j int) bool {
			return parts[i].IsAdmin && !parts[j].IsAdmin
		})
		glib.IdleAdd(func() {
			*currentMembers = parts
			c.populateGroupMembersList(jid, ext, permWidgets, addRow, membersGroup, membersList, parts, reload)
		})
	}()
}
