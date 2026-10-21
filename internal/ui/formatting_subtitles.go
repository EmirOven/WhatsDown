package ui

import (
	"strings"

	"postbetasoftware.com/whatsdown/internal/store"
)

// formatChatSubtitle returns the secondary line for a chat row or dialog.
func formatChatSubtitle(chat store.Chat) string {
	if chat.LastMessageKind != "" || chat.LastMessageText != "" {
		preview := formatMessagePreview(chat.LastMessageKind, chat.LastMessageText)
		isGroup := strings.HasSuffix(chat.JID, "@g.us")

		if isGroup {
			if chat.LastMessageFromMe {
				return "You: " + preview
			}
			sender := chat.LastMessageSender
			if sender != "" && !strings.Contains(sender, "@") {
				return sender + ": " + preview
			}
			return preview
		}
		if chat.LastMessageFromMe {
			return "You: " + preview
		}
		return preview
	}

	displayName := formatDisplayName(chat)
	if strings.HasSuffix(chat.JID, "@g.us") {
		return "Group"
	}
	if strings.HasSuffix(chat.JID, "@broadcast") {
		return "Status"
	}
	if strings.HasSuffix(chat.JID, "@newsletter") {
		return "Channel"
	}
	if strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
		phone := cleanPhoneNumber(chat.JID)
		if phone != "" && phone != displayName {
			return phone
		}
		return ""
	}
	if strings.HasSuffix(chat.JID, "@lid") {
		if strings.HasPrefix(chat.Name, "+") && chat.Name != displayName {
			return chat.Name
		}
		return ""
	}
	return ""
}

// formatTopbarSubtitle returns the secondary line for the active conversation header bar.
func formatTopbarSubtitle(chat store.Chat) string {
	displayName := formatDisplayName(chat)
	if strings.HasSuffix(chat.JID, "@g.us") {
		return "Group"
	}
	if strings.HasSuffix(chat.JID, "@broadcast") {
		return "Status"
	}
	if strings.HasSuffix(chat.JID, "@newsletter") {
		return "Channel"
	}
	if strings.HasSuffix(chat.JID, "@s.whatsapp.net") {
		phone := cleanPhoneNumber(chat.JID)
		if phone != "" && phone != displayName {
			return phone
		}
		return ""
	}
	if strings.HasSuffix(chat.JID, "@lid") {
		if strings.HasPrefix(chat.Name, "+") && chat.Name != displayName {
			return chat.Name
		}
		return ""
	}
	return ""
}
