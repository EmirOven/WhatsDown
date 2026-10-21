package ui

import "go.mau.fi/whatsmeow/proto/waE2E"

// sendPoll builds, sends, and stores a WhatsApp poll.
func (c *MainController) sendPoll(question string, options []string, allowMultiple bool) {
	if c.currentChat == nil {
		return
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}

	selectableCount := 1
	if allowMultiple {
		selectableCount = len(options)
	}

	go c.dispatchPollSend(chatJIDStr, question, options, selectableCount, ctxInfo)
}
