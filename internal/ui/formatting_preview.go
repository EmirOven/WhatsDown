package ui

import (
	"strings"

	"postbetasoftware.com/whatsdown/internal/store"
)

// formatDisplayName returns user-facing title for a chat.
func formatDisplayName(chat store.Chat) string {
	if activeMainWindow != nil && activeMainWindow.isSelfJID(chat.JID) {
		name := strings.TrimSpace(chat.Name)
		if name != "" && !strings.Contains(name, "@") && !strings.Contains(name, "∙∙∙") {
			return name + " (You)"
		}
		return "Message yourself"
	}
	if chat.IsDefaultSubGroup {
		return "Announcements"
	}

	name := strings.TrimSpace(chat.Name)
	if name != "" && !strings.Contains(name, "@") && !strings.Contains(name, "∙∙∙") {
		return name
	}
	switch {
	case strings.HasSuffix(chat.JID, "@s.whatsapp.net"):
		return cleanPhoneNumber(chat.JID)
	case strings.HasSuffix(chat.JID, "@lid"):
		if strings.HasPrefix(name, "+") {
			return name
		}
		return "WhatsApp User"
	case strings.HasSuffix(chat.JID, "@g.us"):
		if name != "" && !strings.Contains(name, "@") {
			return name
		}
		return "Group"
	case strings.HasSuffix(chat.JID, "@broadcast"):
		return "Status Updates"
	case strings.HasSuffix(chat.JID, "@newsletter"):
		if name != "" && !strings.Contains(name, "@") {
			return name
		}
		return "Channel"
	default:
		return cleanPhoneNumber(chat.JID)
	}
}

// formatMessagePreview formats a message into a concise single-line preview snippet.
func formatMessagePreview(kind, text string) string {
	cleanText := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	switch kind {
	case store.MessageKindImage:
		return previewWithFallback("📷 ", cleanText, "📷 Photo")
	case store.MessageKindVideo:
		return previewWithFallback("📹 ", cleanText, "📹 Video")
	case store.MessageKindAudio:
		return "🎵 Audio"
	case store.MessageKindVoice:
		return "🎤 Voice message"
	case store.MessageKindDocument:
		return previewWithFallback("📄 ", cleanText, "📄 Document")
	case store.MessageKindSticker:
		return "Sticker"
	case store.MessageKindLocation:
		return previewWithFallback("📍 ", cleanText, "📍 Location")
	case store.MessageKindContact:
		return previewWithFallback("👤 ", cleanText, "👤 Contact")
	case store.MessageKindPoll:
		return previewWithFallback("📊 ", cleanText, "📊 Poll")
	case store.MessageKindDeleted:
		return "🚫 This message was deleted"
	default:
		if cleanText != "" {
			return cleanText
		}
		return "Message"
	}
}

func previewWithFallback(prefix, text, fallback string) string {
	if text != "" {
		return prefix + text
	}
	return fallback
}
