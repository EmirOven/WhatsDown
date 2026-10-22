package ui

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow/types"
)

type GroupMemberItem struct {
	JID         string
	DisplayName string
	Subtitle    string
}

func (c *MainController) fetchGroupMembers(groupJID string) []GroupMemberItem {
	results := []GroupMemberItem{
		{
			JID:         "all",
			DisplayName: "all",
			Subtitle:    "Notify everyone",
		},
	}
	seen := make(map[string]bool)

	if c.client != nil {
		if jid, err := types.ParseJID(groupJID); err == nil {
			info, err := c.client.GetGroupInfo(context.Background(), jid)
			if err == nil && info != nil {
				for _, p := range info.Participants {
					pTarget := p.JID.String()
					if p.PhoneNumber.User != "" {
						pTarget = p.PhoneNumber.String()
					}
					clean := p.JID.User
					if seen[clean] {
						continue
					}
					seen[clean] = true
					name := c.resolveContactName(pTarget)
					if name == "" || name == "WhatsApp User" {
						name = cleanPhoneNumber(pTarget)
					}
					results = append(results, GroupMemberItem{
						JID:         p.JID.String(),
						DisplayName: name,
						Subtitle:    cleanPhoneNumber(pTarget),
					})
				}
			}
		}
	}

	if len(results) <= 1 && c.store != nil {
		historyMembers := c.getGroupSendersFromHistory(groupJID)
		for _, s := range historyMembers {
			clean := s
			if at := strings.Index(clean, "@"); at != -1 {
				clean = clean[:at]
			}
			if seen[clean] {
				continue
			}
			seen[clean] = true
			name := c.resolveContactName(s)
			results = append(results, GroupMemberItem{
				JID:         s,
				DisplayName: name,
				Subtitle:    cleanPhoneNumber(s),
			})
		}
	}

	return results
}

func (c *MainController) getGroupSendersFromHistory(groupJID string) []string {
	var senders []string
	for _, m := range c.currentMessages {
		if m.SenderJID != "" && !strings.HasPrefix(m.SenderJID, "+") {
			senders = append(senders, m.SenderJID)
		}
	}
	return senders
}
