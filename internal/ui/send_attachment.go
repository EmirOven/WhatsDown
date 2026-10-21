package ui

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"postbetasoftware.com/whatsdown/internal/store"
)

// sendAttachment handles uploading, storing, and displaying outgoing media files.
func (c *MainController) sendAttachment(filePath, caption string) {
	if c.currentChat == nil {
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("reading attachment file: %v", err)
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	filename := filepath.Base(filePath)
	var kind string
	var appMediaType whatsmeow.MediaType
	var mimeType string

	switch ext {
	case ".jpg", ".jpeg":
		kind, appMediaType, mimeType = store.MessageKindImage, whatsmeow.MediaImage, "image/jpeg"
	case ".png":
		kind, appMediaType, mimeType = store.MessageKindImage, whatsmeow.MediaImage, "image/png"
	case ".webp":
		kind, appMediaType, mimeType = store.MessageKindImage, whatsmeow.MediaImage, "image/webp"
	case ".mp4", ".mov", ".mkv", ".webm", ".avi":
		kind, appMediaType, mimeType = store.MessageKindVideo, whatsmeow.MediaVideo, "video/mp4"
	case ".ogg", ".opus", ".mp3", ".m4a", ".wav", ".aac", ".flac":
		kind, appMediaType, mimeType = store.MessageKindAudio, whatsmeow.MediaAudio, "audio/ogg"
		if ext == ".mp3" {
			mimeType = "audio/mpeg"
		} else if ext == ".m4a" {
			mimeType = "audio/mp4"
		}
	default:
		kind, appMediaType, mimeType = store.MessageKindDocument, whatsmeow.MediaDocument, "application/octet-stream"
		if ext == ".pdf" {
			mimeType = "application/pdf"
		}
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	go c.dispatchMediaSend(chatJIDStr, filePath, filename, caption, kind, mimeType, appMediaType, data, ctxInfo)
}
