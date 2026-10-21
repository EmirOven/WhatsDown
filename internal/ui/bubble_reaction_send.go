package ui

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/store"
)

// sendReaction sends or removes a reaction on a message.
func (c *MainController) sendReaction(msg store.StoredMessage, emoji string) {
	chatJID := msg.ChatJID
	if chatJID == "" && c.currentChat != nil {
		chatJID = c.currentChat.JID
	}
	if chatJID == "" {
		return
	}

	ourJID := ""
	if c.client != nil && c.client.Store != nil && c.client.Store.ID != nil {
		ourJID = c.client.Store.ID.ToNonAD().String()
	}

	var myCurrentReaction string
	for _, r := range msg.Reactions {
		if r.FromMe || (ourJID != "" && r.SenderJID == ourJID) || c.isSelfJID(r.SenderJID) {
			myCurrentReaction = r.Emoji
			break
		}
	}

	targetEmoji := emoji
	if emoji != "" && emoji == myCurrentReaction {
		targetEmoji = ""
	}

	if targetEmoji == "" {
		if c.store != nil {
			if ourJID != "" {
				_ = c.store.RemoveReaction(context.Background(), chatJID, msg.ID, ourJID)
			}
			_ = c.store.RemoveMyReaction(context.Background(), chatJID, msg.ID)
		}
	} else if c.store != nil {
		_ = c.store.SaveReaction(context.Background(), store.Reaction{
			ChatJID:    chatJID,
			MessageID:  msg.ID,
			SenderJID:  ourJID,
			SenderName: "You",
			Emoji:      targetEmoji,
			FromMe:     true,
			Timestamp:  time.Now(),
		})
	}

	if c.client != nil && c.client.IsConnected() {
		cChat, err1 := types.ParseJID(chatJID)
		cSender, err2 := types.ParseJID(msg.SenderJID)
		if err1 == nil {
			if msg.FromMe && c.client.Store.ID != nil {
				cSender = c.client.Store.ID.ToNonAD()
			}
			if err2 != nil || cSender.IsEmpty() {
				cSender = cChat
			}
			reactionMsg := c.client.BuildReaction(cChat, cSender, msg.ID, targetEmoji)
			go func() {
				_, err := c.client.SendMessage(context.Background(), cChat, reactionMsg)
				if err != nil {
					log.Printf("error sending reaction: %v", err)
				}
			}()
		}
	}

	glib.IdleAdd(func() {
		c.updateMessageReactions(chatJID, msg.ID)
	})
}
