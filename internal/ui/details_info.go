package ui

import (
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func buildDetailsBadges(chat store.Chat) *gtk.Box {
	chipBox := gtk.NewBox(gtk.OrientationHorizontal, 8)
	chipBox.SetHAlign(gtk.AlignCenter)
	chipBox.SetMarginTop(4)

	chatType := "Direct Message"
	if strings.HasSuffix(chat.JID, "@g.us") {
		chatType = "Group Chat"
	} else if strings.HasSuffix(chat.JID, "@newsletter") {
		chatType = "Channel"
	} else if strings.HasSuffix(chat.JID, "@broadcast") {
		chatType = "Broadcast"
	}
	typeBadge := gtk.NewLabel(chatType)
	typeBadge.AddCSSClass("caption")
	typeBadge.AddCSSClass("card")
	typeBadge.AddCSSClass("status-badge")
	chipBox.Append(typeBadge)

	if chat.Muted {
		muteBadge := gtk.NewLabel("🔇 Muted")
		muteBadge.AddCSSClass("caption")
		muteBadge.AddCSSClass("card")
		muteBadge.AddCSSClass("status-badge")
		chipBox.Append(muteBadge)
	}
	if chat.Pinned {
		pinBadge := gtk.NewLabel("📌 Pinned")
		pinBadge.AddCSSClass("caption")
		pinBadge.AddCSSClass("card")
		pinBadge.AddCSSClass("status-badge")
		chipBox.Append(pinBadge)
	}
	if chat.Archived {
		archBadge := gtk.NewLabel("📁 Archived")
		archBadge.AddCSSClass("caption")
		archBadge.AddCSSClass("card")
		archBadge.AddCSSClass("status-badge")
		chipBox.Append(archBadge)
	}
	return chipBox
}

func (c *MainController) buildDetailsInfoGroup(chat store.Chat, displayName string) *adw.PreferencesGroup {
	infoGroup := adw.NewPreferencesGroup()
	infoGroup.SetTitle("Conversation Details")

	jidRow := adw.NewActionRow()
	if strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
		jidRow.SetTitle("Phone Number")
		jidRow.SetSubtitle(cleanPhoneNumber(chat.JID))
		jidRow.AddPrefix(gtk.NewImageFromIconName("call-start-symbolic"))
	} else if strings.HasSuffix(chat.JID, "@lid") {
		jidRow.SetTitle("Account")
		if strings.HasPrefix(chat.Name, "+") {
			jidRow.SetSubtitle(chat.Name)
		} else {
			jidRow.SetSubtitle("Linked WhatsApp Account")
		}
		jidRow.AddPrefix(gtk.NewImageFromIconName("user-available-symbolic"))
	} else if strings.HasSuffix(chat.JID, "@g.us") {
		jidRow.SetTitle("Chat Type")
		jidRow.SetSubtitle("WhatsApp Group")
		jidRow.AddPrefix(gtk.NewImageFromIconName("system-users-symbolic"))
	} else {
		jidRow.SetTitle("Conversation")
		jidRow.SetSubtitle(displayName)
		jidRow.AddPrefix(gtk.NewImageFromIconName("chat-message-new-symbolic"))
	}
	jidRow.SetSubtitleSelectable(true)
	infoGroup.Add(jidRow)

	if !chat.LastMessageAt.IsZero() {
		lastActiveRow := adw.NewActionRow()
		lastActiveRow.SetTitle("Last Active")
		lastActiveRow.SetSubtitle(chat.LastMessageAt.Format("02 Jan 2006, 15:04"))
		lastActiveRow.AddPrefix(gtk.NewImageFromIconName("document-open-recent-symbolic"))
		infoGroup.Add(lastActiveRow)
	}
	return infoGroup
}
