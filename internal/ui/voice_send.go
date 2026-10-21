package ui

import (
	"log"
	"os"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// sendVoiceMessage uploads and sends an Opus voice message (PTT).
func (c *MainController) sendVoiceMessage(filePath string, duration time.Duration, waveform []byte) {
	if c.currentChat == nil {
		_ = os.Remove(filePath)
		return
	}

	var ctxInfo *waE2E.ContextInfo
	if c.replyingTo != nil {
		ctxInfo = c.buildReplyContextInfo(*c.replyingTo)
		c.cancelReply()
	}

	data, err := os.ReadFile(filePath)
	_ = os.Remove(filePath)
	if err != nil {
		log.Printf("reading voice file: %v", err)
		return
	}

	chatJIDStr := c.currentChat.JID
	if c.isSelfJID(chatJIDStr) {
		if canonical := c.getCanonicalSelfJID(); canonical != "" {
			chatJIDStr = canonical
		}
	}

	seconds := uint32(duration.Seconds())
	if seconds == 0 {
		seconds = 1
	}

	go c.dispatchVoiceSend(chatJIDStr, data, seconds, waveform, ctxInfo)
}
