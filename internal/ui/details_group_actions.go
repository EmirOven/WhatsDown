package ui

import (
	"context"
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/avatar"
)

func (c *MainController) populateGroupMembersList(
	groupJID types.JID, ext *groupExtendedInfo, permWidgets *groupPermWidgets, addRow *adw.ActionRow,
	membersGroup *adw.PreferencesGroup, membersList *gtk.ListBox, parts []types.GroupParticipant, reload func(),
) {
	membersGroup.SetTitle(fmt.Sprintf("Participants (%d)", len(parts)))
	for child := membersList.FirstChild(); child != nil; child = membersList.FirstChild() {
		membersList.Remove(child)
	}

	selfAdmin := false
	for _, p := range parts {
		if (c.isSelfJID(p.JID.String()) || (!p.PhoneNumber.IsEmpty() && c.isSelfJID(p.PhoneNumber.String()))) && (p.IsAdmin || p.IsSuperAdmin) {
			selfAdmin = true
		}
	}
	c.updateGroupPermissionsUI(groupJID, ext, permWidgets, selfAdmin, addRow)
	for _, p := range parts {
		membersList.Append(c.buildGroupMemberRow(groupJID, p, selfAdmin, reload))
	}
}

func (c *MainController) buildGroupMemberRow(groupJID types.JID, p types.GroupParticipant, selfAdmin bool, reload func()) *adw.ActionRow {
	pTarget := p.JID.String()
	if !p.PhoneNumber.IsEmpty() {
		pTarget = p.PhoneNumber.String()
	}
	isSelf := c.isSelfJID(p.JID.String()) || c.isSelfJID(pTarget)
	name := c.resolveContactName(pTarget)
	if isSelf {
		name = "You"
	} else if name == "" || name == "WhatsApp User" {
		name = cleanPhoneNumber(pTarget)
	}

	row := adw.NewActionRow()
	row.SetTitle(name)
	sub := cleanPhoneNumber(pTarget)
	if sub != "" && sub != name {
		row.SetSubtitle(sub)
	}
	av := adw.NewAvatar(36, name, true)
	avatar.LoadAvatar(c.client, pTarget, name, av)
	row.AddPrefix(av)

	if p.IsSuperAdmin || p.IsAdmin {
		lbl := "Group Admin"
		if p.IsSuperAdmin {
			lbl = "Owner"
		}
		badge := gtk.NewLabel(lbl)
		addCSSClasses(badge, "caption", "card", "status-badge")
		badge.SetVAlign(gtk.AlignCenter)
		row.AddSuffix(badge)
	}

	if !isSelf {
		c.attachParticipantContextMenu(row, groupJID, p, selfAdmin, reload)
	}
	return row
}

func (c *MainController) bindAddMembersSubmit(win *gtk.Window, addBtn *gtk.Button, groupJID string, candidates []groupContactItem, onDone func()) {
	addBtn.ConnectClicked(func() {
		var toAdd []types.JID
		for _, cnt := range candidates {
			if cnt.Check != nil && cnt.Check.Active() {
				if jid, err := types.ParseJID(cnt.JID); err == nil {
					toAdd = append(toAdd, jid)
				}
			}
		}
		if len(toAdd) == 0 {
			return
		}
		addBtn.SetSensitive(false)
		go func() {
			if gJID, err := types.ParseJID(groupJID); err == nil {
				_, _ = c.client.UpdateGroupParticipants(context.Background(), gJID, toAdd, whatsmeow.ParticipantChangeAdd)
			}
			glib.IdleAdd(func() { win.Close(); onDone() })
		}()
	})
}
