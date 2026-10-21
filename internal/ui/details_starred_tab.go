package ui

import (
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildDetailsStarredTab(dialog *adw.Dialog, starredMsgs []store.StoredMessage) *gtk.Box {
	starredBox := gtk.NewBox(gtk.OrientationVertical, 8)
	starredBox.SetMarginTop(8)
	if len(starredMsgs) == 0 {
		emptyGroup := adw.NewPreferencesGroup()
		emptyRow := adw.NewActionRow()
		emptyRow.SetTitle("No starred messages")
		emptyRow.SetSubtitle("Right-click any message to star and bookmark it")
		emptyRow.AddPrefix(gtk.NewImageFromIconName("starred-symbolic"))
		emptyGroup.Add(emptyRow)
		starredBox.Append(emptyGroup)
		return starredBox
	}

	starredGroup := adw.NewPreferencesGroup()
	for _, m := range starredMsgs {
		row := adw.NewActionRow()
		titleText := m.Text
		if titleText == "" {
			switch m.Kind {
			case store.MessageKindImage:
				titleText = "📷 Photo"
			case store.MessageKindVideo:
				titleText = "📹 Video"
			case store.MessageKindAudio, store.MessageKindVoice:
				titleText = "🎵 Voice Note"
			case store.MessageKindDocument:
				titleText = "📄 Document"
			default:
				titleText = "💬 Message"
			}
		}
		row.AddPrefix(gtk.NewImageFromIconName("starred-symbolic"))
		row.SetTitle(titleText)
		sub := m.Timestamp.Format("02 Jan 15:04")
		if m.FromMe {
			sub = "You • " + sub
		} else if m.SenderName != "" && !strings.Contains(m.SenderName, "@") {
			sub = m.SenderName + " • " + sub
		}
		row.SetSubtitle(sub)

		jumpBtn := gtk.NewButtonFromIconName("go-jump-symbolic")
		jumpBtn.AddCSSClass("flat")
		jumpBtn.SetTooltipText("Go to Message")
		jumpBtn.SetVAlign(gtk.AlignCenter)
		mCopy := m
		jumpBtn.ConnectClicked(func() {
			dialog.Close()
			c.jumpToMessage(mCopy.ChatJID, mCopy.ID)
		})
		row.AddSuffix(jumpBtn)
		row.SetActivatable(true)
		row.ConnectActivated(func() {
			dialog.Close()
			c.jumpToMessage(mCopy.ChatJID, mCopy.ID)
		})

		c.attachMediaContextMenu(row, mCopy, dialog)
		starredGroup.Add(row)
	}
	starredBox.Append(starredGroup)
	return starredBox
}
