package store

import (
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// DescribeMessage extracts the kind and representative text of a protobuf message.
func DescribeMessage(m *waE2E.Message) (kind, text string, ok bool) {
	switch {
	case m == nil:
		return "", "", false
	case m.GetConversation() != "":
		return MessageKindText, m.GetConversation(), true
	case m.GetExtendedTextMessage() != nil:
		return MessageKindText, m.GetExtendedTextMessage().GetText(), true
	case m.GetImageMessage() != nil:
		return MessageKindImage, m.GetImageMessage().GetCaption(), true
	case m.GetVideoMessage() != nil:
		return MessageKindVideo, m.GetVideoMessage().GetCaption(), true
	case m.GetAudioMessage() != nil:
		if m.GetAudioMessage().GetPTT() {
			return MessageKindVoice, "", true
		}
		return MessageKindAudio, "", true
	case m.GetDocumentMessage() != nil:
		doc := m.GetDocumentMessage()
		text := doc.GetCaption()
		if text == "" {
			text = doc.GetFileName()
		}
		if text == "" {
			text = doc.GetTitle()
		}
		return MessageKindDocument, text, true
	case m.GetStickerMessage() != nil:
		return MessageKindSticker, "", true
	case m.GetLocationMessage() != nil:
		loc := m.GetLocationMessage()
		text := loc.GetName()
		if text == "" {
			text = loc.GetAddress()
		}
		return MessageKindLocation, text, true
	case m.GetLiveLocationMessage() != nil:
		return MessageKindLocation, m.GetLiveLocationMessage().GetCaption(), true
	case m.GetContactMessage() != nil:
		return MessageKindContact, m.GetContactMessage().GetDisplayName(), true
	case m.GetContactsArrayMessage() != nil:
		return MessageKindContact, m.GetContactsArrayMessage().GetDisplayName(), true
	case m.GetCallLogMesssage() != nil:
		cl := m.GetCallLogMesssage()
		text := "Voice call"
		if cl.GetIsVideo() {
			text = "Video call"
		}
		if cl.GetCallOutcome() == waE2E.CallLogMessage_MISSED {
			text = "Missed " + strings.ToLower(text)
		} else if cl.GetDurationSecs() > 0 {
			text = fmt.Sprintf("%s (%s)", text, FormatCallDuration(int(cl.GetDurationSecs())))
		}
		return MessageKindCall, text, true
	}
	for _, poll := range []*waE2E.PollCreationMessage{
		m.GetPollCreationMessage(), m.GetPollCreationMessageV2(), m.GetPollCreationMessageV3(),
		m.GetPollCreationMessageV5(), m.GetPollCreationMessageV6(),
	} {
		if poll != nil {
			return MessageKindPoll, poll.GetName(), true
		}
	}
	return "", "", false
}
