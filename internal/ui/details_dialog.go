package ui

import (
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
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

	headerBox := c.buildDetailsHeader(chat, displayName)
	contentBox.Append(headerBox)
	contentBox.Append(c.buildDetailsInfoGroup(chat, displayName))

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

	phoneSubtitle := formatChatSubtitle(chat)
	if phoneSubtitle == "" && strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
		phoneSubtitle = cleanPhoneNumber(chat.JID)
	}
	if phoneSubtitle != "" && phoneSubtitle != displayName {
		subLabel := gtk.NewLabel(phoneSubtitle)
		subLabel.AddCSSClass("dim-label")
		subLabel.SetHAlign(gtk.AlignCenter)
		headerBox.Append(subLabel)
	}

	headerBox.Append(buildDetailsBadges(chat))
	return headerBox
}
