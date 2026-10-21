package ui

import (
	"log"
	"os"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// sendSticker sends a sticker to the currently active conversation.
func (c *MainController) sendSticker(filePath string) {
	if c.currentChat == nil {
		return
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("reading sticker file %s: %v", filePath, err)
		return
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	go c.dispatchStickerSend(chatJIDStr, data, ctxInfo)
}
