package ui

import (
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

func (c *MainController) prepareMentionsContext(text string, ctxInfo *waE2E.ContextInfo) (string, *waE2E.ContextInfo) {
	if c.currentChat == nil || !strings.HasSuffix(c.currentChat.JID, "@g.us") {
		activeMentionState.insertedMentions = make(map[string]string)
		return text, ctxInfo
	}

	var mentionedJIDs []string
	wireText := text

	if strings.Contains(wireText, "@all") {
		if len(activeMentionState.activeMembers) == 0 {
			activeMentionState.activeMembers = c.fetchGroupMembers(c.currentChat.JID)
		}
		for _, m := range activeMentionState.activeMembers {
			if m.JID != "" && m.JID != "all" {
				mentionedJIDs = append(mentionedJIDs, m.JID)
			}
		}
	}

	for name, jid := range activeMentionState.insertedMentions {
		if jid == "all" {
			continue
		}
		tag := "@" + name
		if strings.Contains(wireText, tag) {
			user := jid
			if idx := strings.Index(user, "@"); idx != -1 {
				user = user[:idx]
			}
			wireText = strings.ReplaceAll(wireText, tag, "@"+user)
			mentionedJIDs = append(mentionedJIDs, jid)
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
