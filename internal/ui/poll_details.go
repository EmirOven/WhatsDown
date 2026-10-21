package ui

import (
	"go.mau.fi/whatsmeow/proto/waE2E"
	"postbetasoftware.com/whatsdown/internal/media"
)

// ReloadPollBubble refreshes the specific poll bubble in the active chat.
func (c *MainController) ReloadPollBubble(pollID string) {
	if c.currentChat == nil {
		return
	}
	c.updateSingleMessageRow(pollID)
}

func (c *MainController) reloadPollBubble(pollID string) {
	c.ReloadPollBubble(pollID)
}

func ExtractPollDetails(raw []byte) (question string, options []string, selectableCount uint32) {
	return extractPollDetailsFromMsg(media.ParseE2EMessage(raw))
}

func extractPollDetails(raw []byte) (question string, options []string, selectableCount uint32) {
	return ExtractPollDetails(raw)
}

func extractPollDetailsFromMsg(e2e *waE2E.Message) (question string, options []string, selectableCount uint32) {
	if e2e == nil {
		return "Poll", nil, 1
	}
	var poll *waE2E.PollCreationMessage
	for _, p := range []*waE2E.PollCreationMessage{
		e2e.GetPollCreationMessage(), e2e.GetPollCreationMessageV2(),
		e2e.GetPollCreationMessageV3(), e2e.GetPollCreationMessageV5(),
		e2e.GetPollCreationMessageV6(),
	} {
		if p != nil {
			poll = p
			break
		}
	}
	if poll == nil {
		return "Poll", nil, 1
	}
	question = poll.GetName()
	if question == "" {
		question = "Poll"
	}
	selectableCount = poll.GetSelectableOptionsCount()
	for _, opt := range poll.GetOptions() {
		options = append(options, opt.GetOptionName())
	}
	return question, options, selectableCount
}
