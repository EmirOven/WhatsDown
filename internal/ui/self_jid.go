package ui

import (
	"context"

	"go.mau.fi/whatsmeow/types"
)

// isSelfJID checks whether the given JID represents the logged-in user.
func (c *MainController) isSelfJID(jid string) bool {
	if jid == "" {
		return false
	}
	if c.store != nil && c.store.IsSelfJID(context.Background(), jid) {
		return true
	}
	if c.client != nil && c.client.Store != nil {
		parsed, err := types.ParseJID(jid)
		if err == nil {
			nonAD := parsed.ToNonAD().String()
			user := parsed.User
			if c.client.Store.ID != nil {
				ownPN := c.client.Store.ID.ToNonAD().String()
				if nonAD == ownPN || user == c.client.Store.ID.User {
					return true
				}
			}
			if !c.client.Store.LID.IsEmpty() {
				ownLID := c.client.Store.LID.ToNonAD().String()
				if nonAD == ownLID || user == c.client.Store.LID.User {
					return true
				}
			}
		}
	}
	return false
}

// getCanonicalSelfJID returns the primary JID for the logged-in user.
func (c *MainController) getCanonicalSelfJID() string {
	if c.store != nil {
		if canonical := c.store.GetCanonicalSelfJID(context.Background()); canonical != "" {
			return canonical
		}
	}
	if c.client != nil && c.client.Store != nil && c.client.Store.ID != nil {
		return c.client.Store.ID.ToNonAD().String()
	}
	return ""
}
