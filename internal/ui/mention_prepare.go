package ui

import (
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

func (c *MainController) prepareMentionsContext(text string, ctxInfo *waE2E.ContextInfo) (string, *waE2E.ContextInfo) {
	if len(activeMentionState.insertedMentions) == 0 {
		return text, ctxInfo
	}

	var mentionedJIDs []string
	wireText := text

	for name, jid := range activeMentionState.insertedMentions {
		tag := "@" + name
		if strings.Contains(wireText, tag) {
			cleanJID := jid
			user := jid
			if idx := strings.Index(user, "@"); idx != -1 {
				user = user[:idx]
			}
			wireTag := "@" + user
			wireText = strings.ReplaceAll(wireText, tag, wireTag)
			mentionedJIDs = append(mentionedJIDs, cleanJID)
		}
	}

	activeMentionState.insertedMentions = make(map[string]string)

	if len(mentionedJIDs) > 0 {
		if ctxInfo == nil {
			ctxInfo = &waE2E.ContextInfo{}
		}
		existing := ctxInfo.GetMentionedJID()
		for _, m := range mentionedJIDs {
			found := false
			for _, e := range existing {
				if e == m {
					found = true
					break
				}
			}
			if !found {
				existing = append(existing, m)
			}
		}
		ctxInfo.MentionedJID = existing
	}

	return wireText, ctxInfo
}
