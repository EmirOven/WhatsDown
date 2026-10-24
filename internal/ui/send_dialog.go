package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/media"
)

// showSendAttachmentDialog presents an in-app preview dialog before sending the attachment.
func (c *MainController) showSendAttachmentDialog(filePath string) {
	if c.currentChat == nil {
		return
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	filename := filepath.Base(filePath)
	fi, err := os.Stat(filePath)
	if err != nil {
		return
	}
	sizeStr := media.FormatFileSize(uint64(fi.Size()))

	dialog := gtk.NewWindow()
	dialog.SetTransientFor(&c.window.Window)
	dialog.SetModal(true)
	dialog.SetDefaultSize(480, 520)
	dialog.SetTitle(attachmentDialogTitle(ext))
	dialog.ConnectCloseRequest(func() bool {
		media.StopCurrentAudio()
		return false
	})

	headerBar := adw.NewHeaderBar()
	cancelBtn := gtk.NewButtonWithLabel("Cancel")
	cancelBtn.AddCSSClass("flat")
	cancelBtn.ConnectClicked(func() {
		media.StopCurrentAudio()
		dialog.Close()
	})
	headerBar.PackStart(cancelBtn)

	sendBtn := gtk.NewButtonWithLabel("Send")
	sendBtn.AddCSSClass("suggested-action")
	headerBar.PackEnd(sendBtn)

	mainBox := gtk.NewBox(gtk.OrientationVertical, 12)
	mainBox.SetMarginTop(12)
	mainBox.SetMarginBottom(16)
	mainBox.SetMarginStart(16)
	mainBox.SetMarginEnd(16)

	mainBox.Append(buildSendPreviewContainer(filePath, ext, filename, sizeStr))

	captionEntry := gtk.NewEntry()
	captionEntry.SetPlaceholderText("Add a caption…")
	captionEntry.SetHExpand(true)
	if c.messageEntry != nil && strings.TrimSpace(c.getMessageText()) != "" {
		captionEntry.SetText(strings.TrimSpace(c.getMessageText()))
	}
	if c.replyingTo != nil {
		preview := gtk.NewLabel("Replying to " + c.replyingTo.SenderName + "\n" + truncatePreviewLines(c.replyingTo.Text, 3))
		if c.replyingTo.Text == "" {
			preview.SetText("Replying to " + c.replyingTo.Kind)
		}
		preview.SetWrap(true)
		preview.SetXAlign(0)
		preview.AddCSSClass("card")
		mainBox.Append(preview)
	}
	mainBox.Append(captionEntry)
	c.attachCaptionMentions(captionEntry, mainBox)

	doSend := func() {
		media.StopCurrentAudio()
		capText := captionEntry.Text()
		dialog.Close()
		c.clearMessageText()
		c.sendAttachment(filePath, capText)
	}
	sendBtn.ConnectClicked(doSend)
	captionEntry.ConnectActivate(doSend)

	dialog.SetTitlebar(headerBar)
	dialog.SetChild(mainBox)
	dialog.Present()
	captionEntry.GrabFocus()
}

func attachmentDialogTitle(ext string) string {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp":
		return "Send Photo"
	case ".mp4", ".mov", ".mkv", ".webm", ".avi":
		return "Send Video"
	case ".mp3", ".ogg", ".opus", ".m4a", ".wav", ".aac", ".flac":
		return "Send Audio"
	case ".pdf":
		return "Send PDF"
	default:
		return "Send Document"
	}
}
