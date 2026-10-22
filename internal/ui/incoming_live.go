package ui

import (
	"context"

	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
)

// HandleIncomingLiveMessage updates the UI when a live message arrives.
func (c *MainController) HandleIncomingLiveMessage(evt *events.Message) {
	if evt.Message != nil && evt.Message.GetPollUpdateMessage() != nil {
		return
	}

	if pm := evt.Message.GetProtocolMessage(); pm != nil {
		targetID := pm.GetKey().GetID()
		if targetID != "" {
			c.updateSingleMessageRow(targetID)
			return
		}
	}

	chatStr := evt.Info.Chat.ToNonAD().String()
	if c.isSelfJID(chatStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatStr = canonical
		}
	} else if c.store != nil {
		if pn := c.store.ResolveLIDtoPN(context.Background(), chatStr); pn != "" {
			chatStr = pn
		}
	}

	isCurrentChat := c.currentChat != nil && c.isSameChat(c.currentChat.JID, chatStr)
	if isCurrentChat {
		var newMsg *store.StoredMessage
		if c.store != nil {
			newMsg, _ = c.store.GetMessage(context.Background(), chatStr, evt.Info.ID)
			if newMsg == nil && c.currentChat != nil {
				newMsg, _ = c.store.GetMessage(context.Background(), c.currentChat.JID, evt.Info.ID)
			}
			if newMsg == nil {
				newMsg, _ = c.store.GetMessageByID(context.Background(), evt.Info.ID)
			}
		}

		if newMsg != nil {
			c.setTypingIndicator(false)
			if _, exists := c.messageRowMap[newMsg.ID]; !exists {
				sameMinuteAsPrev := false
				if len(c.currentMessages) > 0 {
					prevMsg := c.currentMessages[len(c.currentMessages)-1]
					sameMinuteAsPrev = c.isSameAuthorAndDisplayTime(prevMsg, *newMsg)
				}
				c.currentMessages = append(c.currentMessages, *newMsg)
				bubble := c.createMessageBubble(*newMsg, sameMinuteAsPrev)
				if c.messageRowMap != nil {
					c.messageRowMap[newMsg.ID] = bubble
				}
				c.messagesList.Append(bubble)
				c.scrollToBottomAnimated()
			} else {
				c.updateSingleMessageRow(newMsg.ID)
			}
		}
		c.markChatRead(chatStr)
	}

	c.loadChats()
	c.loadArchivedChats()

	if !evt.Info.IsFromMe && !c.isSelfJID(chatStr) {
		isMuted := false
		if c.store != nil {
			if ch, err := c.store.GetChatOrContact(context.Background(), chatStr); err == nil {
				isMuted = ch.Muted
			}
		}
		if !isMuted {
			title, body := c.formatNotificationContent(evt)
			dispatchDesktopNotification(title, body)
		}
	}
}
