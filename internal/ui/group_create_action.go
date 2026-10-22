package ui

import (
	"context"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/avatar"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) bindCreateGroupAction(
	dialog *gtk.Window,
	createBtn *gtk.Button,
	nameEntry, descEntry *gtk.Entry,
	avatarPath *string,
	contacts []groupContactItem,
	errLbl *gtk.Label,
) {
	createBtn.ConnectClicked(func() {
		name := strings.TrimSpace(nameEntry.Text())
		if name == "" {
			errLbl.SetText("Please enter a group name.")
			errLbl.SetVisible(true)
			return
		}
		var participants []types.JID
		for _, cnt := range contacts {
			if cnt.Check != nil && cnt.Check.Active() {
				if parsed, err := types.ParseJID(cnt.JID); err == nil {
					participants = append(participants, parsed)
				}
			}
		}
		if len(participants) == 0 {
			errLbl.SetText("Please select at least 1 member.")
			errLbl.SetVisible(true)
			return
		}

		desc := strings.TrimSpace(descEntry.Text())
		avPath := *avatarPath
		createBtn.SetSensitive(false)
		errLbl.SetVisible(false)

		go c.executeCreateGroup(dialog, createBtn, errLbl, name, desc, avPath, participants)
	})
}

func (c *MainController) executeCreateGroup(
	dialog *gtk.Window, createBtn *gtk.Button, errLbl *gtk.Label, name, desc, avPath string, participants []types.JID,
) {
	ctx := context.Background()
	groupJID := "120363" + time.Now().Format("150405000000") + "@g.us"
	if c.store == nil || !c.store.IsDemo() {
		info, err := c.client.CreateGroup(ctx, whatsmeow.ReqCreateGroup{Name: name, Participants: participants})
		if err != nil || info == nil {
			glib.IdleAdd(func() {
				createBtn.SetSensitive(true)
				errLbl.SetText("Failed to create group: " + err.Error())
				errLbl.SetVisible(true)
			})
			return
		}
		if desc != "" {
			_ = c.client.SetGroupDescription(ctx, info.JID, desc)
		}
		groupJID = info.JID.String()
	}
	if avPath != "" {
		if jpegBytes, err := loadAsJPEG(avPath); err == nil && len(jpegBytes) > 0 {
			if c.store == nil || !c.store.IsDemo() {
				if jid, err := types.ParseJID(groupJID); err == nil {
					_, _ = c.client.SetGroupPhoto(ctx, jid, jpegBytes)
				}
			}
			avatar.StoreBytes(groupJID, jpegBytes)
		}
	}

	now := time.Now()
	if c.store != nil {
		_ = c.store.UpsertChat(ctx, groupJID, name, now.Unix())
	}
	glib.IdleAdd(func() {
		dialog.Close()
		c.loadChats()
		c.selectChat(store.Chat{JID: groupJID, Name: name, LastMessageAt: now})
	})
}
