package markdown

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

var MentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_.-]+)`)

func IsAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func applyMentions(result string, mentionedJIDs []string, resolveName func(string) string) (string, bool) {
	jidMap := make(map[string]string)
	for _, jid := range mentionedJIDs {
		jidMap[jid] = jid
		user := jid
		if idx := strings.Index(user, "@"); idx != -1 {
			user = user[:idx]
		}
		if idx := strings.Index(user, ":"); idx != -1 {
			user = user[:idx]
		}
		if user != "" {
			jidMap[user] = jid
		}
	}

	hasMentions := false
	out := MentionRegex.ReplaceAllStringFunc(result, func(m string) string {
		tag := m[1:]
		if strings.EqualFold(tag, "all") || strings.EqualFold(tag, "everyone") {
			hasMentions = true
			return fmt.Sprintf(`<a href="user:all"><b>@%s</b></a>`, glib.MarkupEscapeText(tag))
		}
		targetJID := tag
		if mapped, ok := jidMap[tag]; ok {
			targetJID = mapped
		} else if !strings.Contains(tag, "@") && IsAllDigits(tag) && len(tag) >= 5 {
			targetJID = tag + "@s.whatsapp.net"
		} else {
			return m
		}

		label := "@" + tag
		if resolveName != nil {
			resolved := strings.TrimPrefix(resolveName(targetJID), "@")
			if resolved != "" && resolved != targetJID && resolved != "WhatsApp User" {
				label = "@" + resolved
			} else if strings.HasPrefix(targetJID, "+") {
				label = "@" + targetJID
			} else if IsAllDigits(tag) && len(tag) >= 5 {
				label = "@+" + tag
			}
		}

		hasMentions = true
		escapedLabel := glib.MarkupEscapeText(label)
		return fmt.Sprintf(`<a href="user:%s"><b>%s</b></a>`, targetJID, escapedLabel)
	})

	return out, hasMentions
}
