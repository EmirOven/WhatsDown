package ui

import (
	"context"

	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
)

// HandleIncomingLiveMessage updates the UI when a live message arrives.
func (c *MainController) HandleIncomingLiveMessage(evt *events.Message) {
	if evt.SourceWebMsg != nil && evt.Info.IsFromMe {
		for view := range c.messageInfoViews {
			if view.messageID == evt.Info.ID && c.isSameChat(view.chatJID, evt.Info.Chat.ToNonAD().String()) {
				view.received()
			}
		}
	}
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
				var previous *store.StoredMessage
				if len(c.currentMessages) > 0 {
					previous = &c.currentMessages[len(c.currentMessages)-1]
				}
				bubble := c.createMessageRow(*newMsg, previous)
				c.currentMessages = append(c.currentMessages, *newMsg)
				if c.messageRowMap != nil {
					c.messageRowMap[newMsg.ID] = bubble
				}
				c.messagesList.Append(bubble)
				c.scrollToBottomAnimated()
			} else {
				c.updateSingleMessageRow(newMsg.ID)
			}
		}
		if c.isWindowFocused() {
			c.markChatRead(chatStr)
		}
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
		if !isMuted && !c.isCurrentChatAndFocused(chatStr) {
			title, body := c.formatNotificationContent(evt)
			c.dispatchDesktopNotification(chatStr, evt.Info.Sender.ToNonAD().String(), evt.Info.ID, title, body)
		}
	}
}
