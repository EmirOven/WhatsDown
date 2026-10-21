package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func buildSendPreviewContainer(filePath, ext, filename, sizeStr string) *gtk.Box {
	previewContainer := gtk.NewBox(gtk.OrientationVertical, 8)
	previewContainer.SetVExpand(true)
	previewContainer.SetHAlign(gtk.AlignCenter)
	previewContainer.SetVAlign(gtk.AlignCenter)

	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp":
		previewContainer.Append(buildImageSendPreview(filePath))
	case ".mp4", ".mov", ".mkv", ".webm", ".avi":
		previewContainer.Append(buildVideoSendPreview(filePath))
	case ".pdf":
		previewContainer.Append(buildPDFSendPreview(filePath))
	case ".mp3", ".ogg", ".opus", ".m4a", ".wav", ".aac", ".flac":
		previewContainer.Append(buildAudioSendPreview(filePath))
	default:
		icon := gtk.NewImageFromIconName("folder-documents-symbolic")
		icon.SetPixelSize(72)
		previewContainer.Append(icon)
	}

	fileInfo := gtk.NewLabel(fmt.Sprintf("%s (%s)", filename, sizeStr))
	addCSSClasses(fileInfo, "caption", "dim-label")
	fileInfo.SetHAlign(gtk.AlignCenter)
	previewContainer.Append(fileInfo)

	return previewContainer
}
