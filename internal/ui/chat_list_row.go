package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"postbetasoftware.com/whatsdown/internal/avatar"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) createChatRow(chat store.Chat, isArchivedView bool) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	row.AddCSSClass("chat-row")

	mainBox := gtk.NewBox(gtk.OrientationHorizontal, 12)
	mainBox.SetHExpand(true)

	if c.store != nil {
		if wp, err := c.store.GetChatWallpaper(context.Background(), chat.JID); err == nil && wp.Kind == "color" && wp.Value != "" {
			bar := gtk.NewBox(gtk.OrientationVertical, 0)
			bar.AddCSSClass("chat-row-color-bar")
			bar.SetVExpand(true)
			prov := gtk.NewCSSProvider()
			prov.LoadFromData(fmt.Sprintf(".chat-row-color-bar { background-color: %s; }", wp.Value))
			bar.StyleContext().AddProvider(prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION+20)
			mainBox.Append(bar)
		}
	}

	displayName := formatDisplayName(chat)
	if strings.HasSuffix(chat.JID, "@g.us") && strings.Contains(chat.LastMessageSender, "@") {
		chat.LastMessageSender = c.resolveContactName(chat.LastMessageSender)
	}
	subText := formatChatSubtitle(chat)

	av := adw.NewAvatar(40, displayName, true)
	av.SetVAlign(gtk.AlignCenter)
	avatar.LoadAvatar(c.client, chat.JID, displayName, av)
	mainBox.Append(av)

	textBox := gtk.NewBox(gtk.OrientationVertical, 4)
	textBox.SetHExpand(true)
	textBox.SetVAlign(gtk.AlignCenter)

	titleLabel := gtk.NewLabel(displayName)
	titleLabel.SetXAlign(0)
	titleLabel.AddCSSClass("chat-row-title")
	titleLabel.SetEllipsize(pango.EllipsizeEnd)
	textBox.Append(titleLabel)

	if subText != "" {
		subLabel := gtk.NewLabel(subText)
		subLabel.SetXAlign(0)
		subLabel.AddCSSClass("chat-row-subtitle")
		subLabel.SetEllipsize(pango.EllipsizeEnd)
		subLabel.SetSingleLineMode(true)
		textBox.Append(subLabel)
	}
	mainBox.Append(textBox)

	metaBox := gtk.NewBox(gtk.OrientationVertical, 4)
	metaBox.SetVAlign(gtk.AlignStart)

	if !chat.LastMessageAt.IsZero() {
		timeLabel := gtk.NewLabel(formatChatTime(chat.LastMessageAt))
		timeLabel.SetXAlign(1)
		timeLabel.AddCSSClass("chat-row-time")
		metaBox.Append(timeLabel)
	}

	subMetaBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	subMetaBox.SetHAlign(gtk.AlignEnd)

	if chat.Muted {
		muteIcon := gtk.NewImageFromIconName("audio-volume-muted-symbolic")
		muteIcon.AddCSSClass("pin-indicator")
		subMetaBox.Append(muteIcon)
	}
	if chat.Pinned {
		pinIcon := gtk.NewImageFromIconName("view-pin-symbolic")
		pinIcon.AddCSSClass("pin-indicator")
		subMetaBox.Append(pinIcon)
	}
	if chat.UnreadCount > 0 {
		badge := gtk.NewLabel(fmt.Sprintf("%d", chat.UnreadCount))
		badge.AddCSSClass("unread-badge")
		badge.SetXAlign(0.5)
		subMetaBox.Append(badge)
	}

	metaBox.Append(subMetaBox)
	mainBox.Append(metaBox)
	row.SetChild(mainBox)

	c.attachContextMenu(row, chat, isArchivedView)
	return row
}
