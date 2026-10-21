package ui

import (
	"context"
	"log"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
)

// startEditing populates the messageEntry and reveals the edit banner.
func (c *MainController) startEditing(msg store.StoredMessage) {
	c.cancelReply()
	mCopy := msg
	c.editingMessage = &mCopy
	if c.editBannerText != nil {
		c.editBannerText.SetText(msg.Text)
	}
	if c.editBannerBar != nil {
		c.editBannerBar.SetVisible(true)
	}
	c.scrollToBottomAnimated()
	if c.messageEntry != nil {
		c.setMessageText(msg.Text)
		c.messageEntry.GrabFocus()
	}
}

// cancelEditing clears editing state and hides the edit banner.
func (c *MainController) cancelEditing() {
	c.editingMessage = nil
	if c.editBannerBar != nil {
		c.editBannerBar.SetVisible(false)
	}
	c.scrollToBottomAnimated()
	c.clearMessageText()
}

// submitEdit updates the message in local storage and sends edit protocol message via WhatsApp.
func (c *MainController) submitEdit(msg store.StoredMessage, newText string) {
	c.cancelEditing()
	if strings.TrimSpace(newText) == "" || newText == msg.Text {
		return
	}

	if c.store != nil {
		if err := c.store.SaveMessageEdit(context.Background(), msg.ChatJID, msg.ID, newText); err != nil {
			log.Printf("saving message edit: %v", err)
		}
	}

	if c.client != nil {
		targetJID, err := types.ParseJID(msg.ChatJID)
		if err == nil {
			editMsg := c.client.BuildEdit(targetJID, msg.ID, &waE2E.Message{
				Conversation: proto.String(newText),
			})
			go func() {
				_, err := c.client.SendMessage(context.Background(), targetJID, editMsg)
				if err != nil {
					log.Printf("sending edit message: %v", err)
				}
			}()
		}
	}

	if c.currentChat != nil && c.isSameChat(c.currentChat.JID, msg.ChatJID) {
		c.updateSingleMessageRow(msg.ID)
	}
	c.loadChats()
}
