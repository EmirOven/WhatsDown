package ui

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/avatar"
)

var DatabasePathFunc func() (string, error)

func (c *MainController) refreshSettingsInfo() {
	if c.client != nil && c.client.Store != nil && c.client.Store.ID != nil {
		pushName := c.client.Store.PushName
		if pushName == "" {
			pushName = "WhatsApp User"
		}
		phone := "+" + c.client.Store.ID.User
		if c.settingsProfileRow != nil {
			c.settingsProfileRow.SetTitle(pushName)
			c.settingsProfileRow.SetSubtitle(phone)
		}
		ownJID := c.client.Store.ID.ToNonAD().String()
		if c.settingsProfileAvatar != nil {
			avatar.LoadAvatar(c.client, ownJID, pushName, c.settingsProfileAvatar)
		}
	}

	if DatabasePathFunc != nil {
		if dbPath, err := DatabasePathFunc(); err == nil && c.settingsDbPathRow != nil {
			c.settingsDbPathRow.SetSubtitle(dbPath)
		}
	} else if c.settingsDbPathRow != nil {
		dir := filepath.Join(glib.GetUserDataDir(), "whatsdown")
		c.settingsDbPathRow.SetSubtitle(filepath.Join(dir, "whatsdown.db"))
	}

	if c.store != nil && c.settingsStatsRow != nil {
		go func() {
			chats, msgs, _ := c.store.GetStats(context.Background())
			glib.IdleAdd(func() {
				if c.settingsStatsRow != nil {
					c.settingsStatsRow.SetSubtitle(fmt.Sprintf("%d chats • %d messages", chats, msgs))
				}
			})
		}()
	}
}
