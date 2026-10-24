package ui

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) buildDetailsAudioTab(dialog *adw.Dialog, audioMsgs []store.StoredMessage) *gtk.Box {
	audioBox := gtk.NewBox(gtk.OrientationVertical, 8)
	audioBox.SetMarginTop(8)
	if len(audioMsgs) == 0 {
		emptyGroup := adw.NewPreferencesGroup()
		emptyRow := adw.NewActionRow()
		emptyRow.SetTitle("No audio files")
		emptyRow.SetSubtitle("Voice notes and audio files will appear here")
		emptyRow.AddPrefix(gtk.NewImageFromIconName("folder-music-symbolic"))
		emptyGroup.Add(emptyRow)
		audioBox.Append(emptyGroup)
		return audioBox
	}

	audioGroup := adw.NewPreferencesGroup()
	for _, m := range audioMsgs {
		row := adw.NewActionRow()
		titleText := "Audio Message"
		if m.Kind == store.MessageKindVoice {
			titleText = "Voice Note"
		}
		e2e := media.ParseE2EMessage(m.Raw)
		if e2e != nil && e2e.GetAudioMessage() != nil {
			if s := e2e.GetAudioMessage().GetSeconds(); s > 0 {
				titleText += fmt.Sprintf(" (%ds)", s)
			}
		}
		row.AddPrefix(gtk.NewImageFromIconName("folder-music-symbolic"))
		row.SetTitle(titleText)
		sub := m.Timestamp.Format("02 Jan 15:04")
		if m.FromMe {
			sub = "You • " + sub
		} else if m.SenderName != "" && !strings.Contains(m.SenderName, "@") {
			sub = m.SenderName + " • " + sub
		}
		row.SetSubtitle(sub)

		mCopy := m
		row.AddSuffix(c.buildAudioWidget(mCopy))

		c.attachMediaContextMenu(row, mCopy, dialog)
		audioGroup.Add(row)
	}
	audioBox.Append(audioGroup)
	dialog.ConnectClosed(func() { c.releaseMessageWidget(audioBox) })
	return audioBox
}
