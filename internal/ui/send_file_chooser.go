package ui

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// openFileChooserForMedia displays a native file chooser for attaching media or documents.
func (c *MainController) openFileChooserForMedia(imagesAndVideosOnly bool) {
	title := "Choose Document to Send"
	if imagesAndVideosOnly {
		title = "Choose Photo or Video to Send"
	}

	dialog := gtk.NewFileChooserNative(title, &c.window.Window, gtk.FileChooserActionOpen, "_Select", "_Cancel")
	filter := gtk.NewFileFilter()
	if imagesAndVideosOnly {
		filter.SetName("Photos & Videos")
		filter.AddMIMEType("image/*")
		filter.AddMIMEType("video/*")
		filter.AddPattern("*.jpg")
		filter.AddPattern("*.jpeg")
		filter.AddPattern("*.png")
		filter.AddPattern("*.webp")
		filter.AddPattern("*.gif")
		filter.AddPattern("*.mp4")
		filter.AddPattern("*.mov")
		filter.AddPattern("*.mkv")
		filter.AddPattern("*.webm")
	} else {
		filter.SetName("All Documents & Files")
		filter.AddMIMEType("application/pdf")
		filter.AddMIMEType("application/*")
		filter.AddMIMEType("text/*")
		filter.AddPattern("*")
	}
	dialog.SetFilter(filter)

	dialog.ConnectResponse(func(responseId int) {
		if responseId == int(gtk.ResponseAccept) {
			if file := dialog.File(); file != nil {
				if path := file.Path(); path != "" {
					glib.IdleAdd(func() {
						c.showSendAttachmentDialog(path)
					})
				}
			}
		}
	})
	dialog.Show()
}
