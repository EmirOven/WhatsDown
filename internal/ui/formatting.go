package ui

import (
	"strings"
	"time"

	"postbetasoftware.com/whatsdown/internal/store"
)

// cleanPhoneNumber formats a phone number cleanly with a leading '+',
// stripping device indices and domain suffixes.
func cleanPhoneNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "@lid") {
		return "WhatsApp User"
	}
	if idx := strings.Index(raw, "@"); idx != -1 {
		raw = raw[:idx]
	}
	if idx := strings.Index(raw, ":"); idx != -1 {
		raw = raw[:idx]
	}
	if raw == "" {
		return ""
	}
	allDigits := true
	for _, r := range raw {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if allDigits && len(raw) >= 5 {
		if len(raw) == 15 && strings.HasPrefix(raw, "1") {
			return "WhatsApp User"
		}
		return "+" + raw
	}
	return raw
}

// formatStatusMarkup returns Pango markup representing delivery and read ticks.
func formatStatusMarkup(status int) string {
	switch status {
	case store.MessageStatusRead:
		return `<span foreground="#53bdeb" font_weight="bold">・・</span>`
	case store.MessageStatusDelivered:
		return `<span alpha="85%">・・</span>`
	case store.MessageStatusSent:
		return `<span alpha="70%">・</span>`
	default:
		return `<span foreground="#53bdeb" font_weight="bold">・・</span>`
	}
}

// formatChatTime formats a timestamp for the chat list row.
func formatChatTime(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	if now.Sub(t) < 7*24*time.Hour {
		return t.Format("Mon")
	}
	return t.Format("02/01/06")
}
