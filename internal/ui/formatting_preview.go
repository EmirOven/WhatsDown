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

	name := strings.TrimSpace(chat.Name)
	if name != "" && !strings.Contains(name, "@") && !strings.Contains(name, "∙∙∙") {
		return name
	}
	if strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
		return cleanPhoneNumber(chat.JID)
	}
	if strings.HasSuffix(chat.JID, "@lid") {
		if strings.HasPrefix(name, "+") {
			return name
		}
		return "WhatsApp User"
	}
	if strings.HasSuffix(chat.JID, "@g.us") {
		if name != "" && !strings.Contains(name, "@") {
			return name
		}
		return "Group"
	}
	if strings.HasSuffix(chat.JID, "@broadcast") {
		return "Status Updates"
	}
	if strings.HasSuffix(chat.JID, "@newsletter") {
		if name != "" && !strings.Contains(name, "@") {
			return name
		}
		return "Channel"
	}
	return cleanPhoneNumber(chat.JID)
}

// formatMessagePreview formats a message into a concise single-line preview snippet.
func formatMessagePreview(kind, text string) string {
	cleanText := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	switch kind {
	case store.MessageKindImage:
		if cleanText != "" {
			return "📷 " + cleanText
		}
		return "📷 Photo"
	case store.MessageKindVideo:
		if cleanText != "" {
			return "📹 " + cleanText
		}
		return "📹 Video"
	case store.MessageKindAudio:
		return "🎵 Audio"
	case store.MessageKindVoice:
		return "🎤 Voice message"
	case store.MessageKindDocument:
		if cleanText != "" {
			return "📄 " + cleanText
		}
		return "📄 Document"
	case store.MessageKindSticker:
		return "Sticker"
	case store.MessageKindLocation:
		if cleanText != "" {
			return "📍 " + cleanText
		}
		return "📍 Location"
	case store.MessageKindContact:
		if cleanText != "" {
			return "👤 " + cleanText
		}
		return "👤 Contact"
	case store.MessageKindPoll:
		if cleanText != "" {
			return "📊 " + cleanText
		}
		return "📊 Poll"
	case store.MessageKindDeleted:
		return "🚫 This message was deleted"
	default:
		if cleanText != "" {
			return cleanText
		}
		return "Message"
	}
}
