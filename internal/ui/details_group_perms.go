package ui

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

type groupPermWidgets struct {
	group                                                  *adw.PreferencesGroup
	editSwitch, msgSwitch, addSwitch, linkSwitch, histSwitch *gtk.Switch
	bound, updating                                        bool
}

func buildGroupPermissionsGroup() *groupPermWidgets {
	grp := adw.NewPreferencesGroup()
	grp.SetTitle("Group Permissions")
	mkRow := func(title, sub string) *gtk.Switch {
		row := adw.NewActionRow()
		row.SetTitle(title)
		row.SetSubtitle(sub)
		sw := gtk.NewSwitch()
		sw.SetVAlign(gtk.AlignCenter)
		sw.SetSensitive(false)
		row.AddSuffix(sw)
		row.SetActivatableWidget(sw)
		grp.Add(row)
		return sw
	}
	return &groupPermWidgets{
		group:      grp,
		editSwitch: mkRow("Edit group settings", "Allow all members to change name, icon and description"),
		msgSwitch:  mkRow("Admin-only messages", "Only admins can send messages in this group"),
		addSwitch:  mkRow("Add other members", "Allow all members to add people to this group"),
		linkSwitch: mkRow("Invite via link or QR code", "Allow members to share invite links without admin approval"),
		histSwitch: mkRow("Send message history", "Share recent messages with newly added members"),
	}
}

func (c *MainController) updateGroupPermissionsUI(groupJID types.JID, info *types.GroupInfo, pw *groupPermWidgets, isAdmin bool, addRow *adw.ActionRow) {
	addRow.SetSensitive(isAdmin || info.MemberAddMode == types.GroupMemberAddModeAllMember)
	pw.updating = true
	pw.editSwitch.SetActive(!info.IsLocked)
	pw.msgSwitch.SetActive(info.IsAnnounce)
	pw.addSwitch.SetActive(info.MemberAddMode == types.GroupMemberAddModeAllMember)
	pw.linkSwitch.SetActive(!info.IsJoinApprovalRequired)
	pw.updating = false
	for _, sw := range []*gtk.Switch{pw.editSwitch, pw.msgSwitch, pw.addSwitch, pw.linkSwitch, pw.histSwitch} {
		sw.SetSensitive(isAdmin)
	}
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
		mode := types.GroupMemberAddModeAdmin
		if s {
			mode = types.GroupMemberAddModeAllMember
		}
		err := c.client.SetGroupMemberAddMode(context.Background(), groupJID, mode)
		if err == nil {
			glib.IdleAdd(func() { addRow.SetSensitive(isAdmin || s) })
		}
		return err
	})
	bindSw(pw.linkSwitch, func(s bool) error { return c.client.SetGroupJoinApprovalMode(context.Background(), groupJID, !s) })
	bindSw(pw.histSwitch, func(s bool) error { return c.setGroupHistoryShareMode(groupJID, s) })
}

func (c *MainController) setGroupHistoryShareMode(groupJID types.JID, enabled bool) error {
	mode := "admin_share"
	if enabled {
		mode = "all_member_share"
	}
	_, err := c.client.DangerousInternals().SendGroupIQ(context.Background(), "set", groupJID, waBinary.Node{
		Tag: "member_share_group_history_mode", Content: []byte(mode),
	})
	return err
}
