package ui

import (
	"log"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// sendMessage sends the text from messageEntry to the currently selected chat.
func (c *MainController) sendMessage() {
	if c.currentChat == nil || c.client == nil {
		return
	}

	text := strings.TrimSpace(c.getMessageText())
	if text == "" {
		return
	}
	c.clearMessageText()
	if c.markdownPreviewBar != nil {
		c.markdownPreviewBar.SetVisible(false)
	}

	if c.editingMessage != nil {
		targetMsg := *c.editingMessage
		c.submitEdit(targetMsg, text)
		return
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	text, ctxInfo = c.prepareMentionsContext(text, ctxInfo)

	targetJID, err := types.ParseJID(c.currentChat.JID)
	if err != nil {
		log.Printf("parsing target JID: %v", err)
		return
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}
	c.autoScrollBottom = true
	c.scrollToBottomAnimated()

	go c.dispatchTextMessage(targetJID, chatJIDStr, text, ctxInfo)
}
