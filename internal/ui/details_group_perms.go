package ui

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/types"
)

type groupPermWidgets struct {
	group                                                                 *adw.PreferencesGroup
	linkRow                                                               *adw.ActionRow
	editSwitch, msgSwitch, addSwitch, linkSwitch, histSwitch, approveSwitch *gtk.Switch
	bound, updating                                                       bool
}

func buildGroupPermissionsGroup() *groupPermWidgets {
	grp := adw.NewPreferencesGroup()
	grp.SetTitle("Group Permissions")
	mkRow := func(title, sub string) (*adw.ActionRow, *gtk.Switch) {
		row := adw.NewActionRow()
		row.SetTitle(title)
		row.SetSubtitle(sub)
		sw := gtk.NewSwitch()
		sw.SetVAlign(gtk.AlignCenter)
		sw.SetSensitive(false)
		row.AddSuffix(sw)
		row.SetActivatableWidget(sw)
		grp.Add(row)
		return row, sw
	}
	_, editSw := mkRow("Edit group settings", "Allow all members to change name, icon and description")
	_, msgSw := mkRow("Admin-only messages", "Only admins can send messages in this group")
	_, addSw := mkRow("Add other members", "Allow all members to add people to this group")
	linkRow, linkSw := mkRow("Invite via link or QR code", "Allow members to share invite links and QR codes")
	_, histSw := mkRow("Send message history", "Share recent messages with newly added members")
	_, appSw := mkRow("Approve new members", "Require an admin to approve new members before joining")
	linkRow.SetVisible(false)
	return &groupPermWidgets{
		group: grp, linkRow: linkRow, editSwitch: editSw, msgSwitch: msgSw,
		addSwitch: addSw, linkSwitch: linkSw, histSwitch: histSw, approveSwitch: appSw,
	}
}

func (c *MainController) updateGroupPermissionsUI(groupJID types.JID, ext *groupExtendedInfo, pw *groupPermWidgets, isAdmin bool, addRow *adw.ActionRow) {
	info := ext.Info
	addEnabled := info.MemberAddMode == types.GroupMemberAddModeAllMember
	addRow.SetSensitive(isAdmin || addEnabled)
	pw.updating = true
	pw.editSwitch.SetActive(!info.IsLocked)
	pw.msgSwitch.SetActive(info.IsAnnounce)
	pw.addSwitch.SetActive(addEnabled)
	pw.linkSwitch.SetActive(ext.AllMemberLink)
	pw.histSwitch.SetActive(ext.HistoryShareAll || ext.GroupHistoryToggle)
	pw.approveSwitch.SetActive(info.IsJoinApprovalRequired)
	pw.updating = false
	pw.linkRow.SetVisible(addEnabled)
	for _, sw := range []*gtk.Switch{pw.editSwitch, pw.msgSwitch, pw.addSwitch, pw.histSwitch, pw.approveSwitch} {
		sw.SetSensitive(isAdmin)
	}
	pw.linkSwitch.SetSensitive(isAdmin && addEnabled)
	if pw.bound {
		return
	}
	pw.bound = true
	bindSw := func(sw *gtk.Switch, fn func(bool) error) {
		sw.ConnectStateSet(func(state bool) bool {
			if !pw.updating && c.client != nil {
				go func() {
					if fn(state) != nil {
						glib.IdleAdd(func() { pw.updating = true; sw.SetActive(!state); pw.updating = false })
					}
				}()
			}
			return false
		})
	}
	bindSw(pw.editSwitch, func(s bool) error { return c.client.SetGroupLocked(context.Background(), groupJID, !s) })
	bindSw(pw.msgSwitch, func(s bool) error { return c.client.SetGroupAnnounce(context.Background(), groupJID, s) })
	bindSw(pw.addSwitch, func(s bool) error {
		setDeps := func(on bool) { pw.linkRow.SetVisible(on); pw.linkSwitch.SetSensitive(isAdmin && on) }
		glib.IdleAdd(func() { setDeps(s) })
		mode := map[bool]types.GroupMemberAddMode{false: types.GroupMemberAddModeAdmin, true: types.GroupMemberAddModeAllMember}[s]
		err := c.client.SetGroupMemberAddMode(context.Background(), groupJID, mode)
		glib.IdleAdd(func() {
			if err != nil {
				setDeps(!s)
			} else {
				addRow.SetSensitive(isAdmin || s)
			}
		})
		return err
	})
	bindSw(pw.linkSwitch, func(s bool) error { return c.setGroupLinkMode(context.Background(), groupJID, s) })
	bindSw(pw.histSwitch, func(s bool) error { return c.setGroupHistoryShareMode(context.Background(), groupJID, s) })
	bindSw(pw.approveSwitch, func(s bool) error { return c.client.SetGroupJoinApprovalMode(context.Background(), groupJID, s) })
}
