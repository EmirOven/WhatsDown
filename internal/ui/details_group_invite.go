package ui

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow/types"
)

func (c *MainController) addGroupInviteActionRows(actionsGroup *adw.PreferencesGroup, groupJID string, pw *groupPermWidgets) {
	linkRow := adw.NewActionRow()
	linkRow.SetTitle("Copy Invite Link")
	linkRow.SetSubtitle("Generate and copy group invite link to clipboard")
	linkRow.AddPrefix(gtk.NewImageFromIconName("insert-link-symbolic"))
	linkRow.SetActivatable(true)
	linkRow.SetVisible(false)
	linkRow.ConnectActivated(func() { c.copyGroupInviteLink(groupJID, linkRow) })
	actionsGroup.Add(linkRow)

	qrRow := adw.NewActionRow()
	qrRow.SetTitle("Copy Invite QR Code")
	qrRow.SetSubtitle("Generate and copy group invite QR code to clipboard")
	qrRow.AddPrefix(gtk.NewImageFromIconName("qr-code-symbolic"))
	qrRow.SetActivatable(true)
	qrRow.SetVisible(false)
	qrRow.ConnectActivated(func() { c.copyGroupInviteQR(groupJID, qrRow) })
	actionsGroup.Add(qrRow)

	pw.copyLinkRow = linkRow
	pw.copyQRRow = qrRow
}

func (pw *groupPermWidgets) updateInviteButtonsVisibility(isAdmin bool) {
	canInvite := isAdmin || (pw.addSwitch.Active() && pw.linkSwitch.Active())
	for _, r := range []*adw.ActionRow{pw.copyLinkRow, pw.copyQRRow} {
		if r != nil {
			r.SetVisible(canInvite)
			r.SetSensitive(canInvite)
		}
	}
}

func (c *MainController) resolveGroupInviteLink(groupJID string) string {
	if c.store != nil && c.store.IsDemo() {
		return "https://chat.whatsapp.com/DemoInviteLinkWhatsDown"
	}
	if c.client == nil || !c.client.IsConnected() {
		return ""
	}
	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return ""
	}
	link, _ := c.client.GetGroupInviteLink(context.Background(), jid, false)
	return link
}

func (c *MainController) copyGroupInviteLink(groupJID string, row *adw.ActionRow) {
	go func() {
		link := c.resolveGroupInviteLink(groupJID)
		if link == "" {
			return
		}
		glib.IdleAdd(func() {
			if disp := gdk.DisplayGetDefault(); disp != nil {
				disp.Clipboard().SetText(link)
			}
			row.SetSubtitle("Copied to clipboard: " + link)
		})
	}()
}

func (c *MainController) copyGroupInviteQR(groupJID string, row *adw.ActionRow) {
	go func() {
		link := c.resolveGroupInviteLink(groupJID)
		if link == "" {
			return
		}
		pngBytes, err := qrcode.Encode(link, qrcode.Medium, 512)
		if err != nil {
			return
		}
		glib.IdleAdd(func() {
			if tex, err := gdk.NewTextureFromBytes(glib.NewBytesWithGo(pngBytes)); err == nil && tex != nil {
				if disp := gdk.DisplayGetDefault(); disp != nil {
					disp.Clipboard().SetTexture(tex)
				}
				row.SetSubtitle("QR code copied to clipboard!")
			}
		})
	}()
}
