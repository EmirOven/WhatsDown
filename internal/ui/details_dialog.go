package ui

import (
	"context"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/avatar"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) showDetailsAndMedia(chat store.Chat) {
	dialog := adw.NewDialog()
	displayName := formatDisplayName(chat)
	dialog.SetTitle(displayName)
	dialog.SetContentWidth(460)
	dialog.SetContentHeight(600)

	toolbarView := adw.NewToolbarView()
	toolbarView.AddTopBar(adw.NewHeaderBar())

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetVExpand(true)

	contentBox := gtk.NewBox(gtk.OrientationVertical, 16)
	contentBox.SetMarginTop(18)
	contentBox.SetMarginBottom(24)
	contentBox.SetMarginStart(18)
	contentBox.SetMarginEnd(18)
	contentBox.Append(c.buildDetailsHeader(chat, displayName))
	contentBox.Append(c.buildDetailsInfoGroup(chat, displayName))
	if strings.HasSuffix(chat.JID, "@g.us") {
		c.appendGroupManagementSections(dialog, chat, contentBox)
	}

	if c.store != nil {
		contentBox.Append(c.buildDetailsWallpaperGroup(dialog, chat))
		c.buildDetailsTabs(dialog, chat, contentBox)
	}

	scrolled.SetChild(contentBox)
	toolbarView.SetContent(scrolled)
	dialog.SetChild(toolbarView)
	dialog.Present(c.window)
}

func (c *MainController) buildDetailsHeader(chat store.Chat, displayName string) *gtk.Box {
	headerBox := gtk.NewBox(gtk.OrientationVertical, 8)
	headerBox.SetHAlign(gtk.AlignCenter)

	av := adw.NewAvatar(80, displayName, true)
	av.SetHAlign(gtk.AlignCenter)
	avatar.LoadAvatar(c.client, chat.JID, displayName, av)
	headerBox.Append(av)

	nameLabel := gtk.NewLabel(displayName)
	nameLabel.AddCSSClass("title-1")
	nameLabel.SetHAlign(gtk.AlignCenter)
	nameLabel.SetWrap(true)
	nameLabel.SetWrapMode(pango.WrapWordChar)
	headerBox.Append(nameLabel)

	c.appendDetailsSubtitle(headerBox, chat, displayName)
	headerBox.Append(buildDetailsBadges(chat))
	return headerBox
}

func (c *MainController) appendDetailsSubtitle(headerBox *gtk.Box, chat store.Chat, displayName string) {
	subLabel := gtk.NewLabel("")
	addCSSClasses(subLabel, "dim-label")
	subLabel.SetHAlign(gtk.AlignCenter)
	subLabel.SetWrap(true)
	subLabel.SetWrapMode(pango.WrapWordChar)
	subLabel.SetJustify(gtk.JustifyCenter)
	subLabel.SetVisible(false)
	headerBox.Append(subLabel)

	if strings.HasSuffix(chat.JID, "@g.us") && c.client != nil && c.client.IsConnected() {
		go func() {
			if jid, err := types.ParseJID(chat.JID); err == nil {
				if info, err := c.client.GetGroupInfo(context.Background(), jid); err == nil && info != nil && strings.TrimSpace(info.Topic) != "" {
					topic := strings.TrimSpace(info.Topic)
					glib.IdleAdd(func() { subLabel.SetText(topic); subLabel.SetVisible(true) })
				}
			}
		}()
	} else if !strings.HasSuffix(chat.JID, "@g.us") {
		if sub := formatTopbarSubtitle(chat); sub != "" && sub != displayName {
			subLabel.SetText(sub)
			subLabel.SetVisible(true)
		}
	}
}
