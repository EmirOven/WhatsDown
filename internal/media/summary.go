package media

import (
	"go.mau.fi/whatsmeow/proto/waE2E"
	"postbetasoftware.com/whatsdown/internal/store"
)

// SummarizeProtoMessage extracts representative text and kind from a protobuf message.
func SummarizeProtoMessage(waMsg *waE2E.Message) (string, string) {
	if waMsg == nil {
		return "Message", store.MessageKindText
	}
	if text := waMsg.GetConversation(); text != "" {
		return text, store.MessageKindText
	}
	if ext := waMsg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText(), store.MessageKindText
	}
	if img := waMsg.GetImageMessage(); img != nil {
		if cap := img.GetCaption(); cap != "" {
			return "📷 " + cap, store.MessageKindImage
		}
		return "📷 Photo", store.MessageKindImage
	}
	if vid := waMsg.GetVideoMessage(); vid != nil {
		if cap := vid.GetCaption(); cap != "" {
			return "🎥 " + cap, store.MessageKindVideo
		}
		return "🎥 Video", store.MessageKindVideo
	}
	if aud := waMsg.GetAudioMessage(); aud != nil {
		if aud.GetPTT() {
			return "🎤 Voice message", store.MessageKindVoice
		}
		return "🎵 Audio", store.MessageKindAudio
	}
	if doc := waMsg.GetDocumentMessage(); doc != nil {
		if fn := doc.GetFileName(); fn != "" {
			return "📄 " + fn, store.MessageKindDocument
		}
		return "📄 Document", store.MessageKindDocument
	}
	if waMsg.GetLocationMessage() != nil || waMsg.GetLiveLocationMessage() != nil {
		return "📍 Location", store.MessageKindLocation
	}
	if waMsg.GetPollCreationMessage() != nil || waMsg.GetPollCreationMessageV2() != nil || waMsg.GetPollCreationMessageV3() != nil {
		_, text, _ := store.DescribeMessage(waMsg)
		if text != "" {
			return "📊 " + text, store.MessageKindPoll
		}
		return "📊 Poll", store.MessageKindPoll
	}
	if waMsg.GetStickerMessage() != nil {
		return "🏷️ Sticker", store.MessageKindSticker
	}
	if waMsg.GetContactMessage() != nil || waMsg.GetContactsArrayMessage() != nil {
		return "👤 Contact", store.MessageKindText
	}
	return "Message", store.MessageKindText
}
