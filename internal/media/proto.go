package media

import (
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

// ParseE2EMessage parses raw protobuf bytes into a waE2E.Message, unwrapping wrappers.
func ParseE2EMessage(raw []byte) *waE2E.Message {
	if len(raw) == 0 {
		return nil
	}
	var msg waE2E.Message
	if err := proto.Unmarshal(raw, &msg); err != nil {
		return nil
	}
	return UnwrapE2EMessage(&msg)
}

// UnwrapE2EMessage unwraps ephemeral, view-once, and document wrappers.
func UnwrapE2EMessage(msg *waE2E.Message) *waE2E.Message {
	if msg == nil {
		return nil
	}
	if ephem := msg.GetEphemeralMessage(); ephem != nil && ephem.GetMessage() != nil {
		return UnwrapE2EMessage(ephem.GetMessage())
	}
	if vo := msg.GetViewOnceMessage(); vo != nil && vo.GetMessage() != nil {
		return UnwrapE2EMessage(vo.GetMessage())
	}
	if vo2 := msg.GetViewOnceMessageV2(); vo2 != nil && vo2.GetMessage() != nil {
		return UnwrapE2EMessage(vo2.GetMessage())
	}
	if vo2Ext := msg.GetViewOnceMessageV2Extension(); vo2Ext != nil && vo2Ext.GetMessage() != nil {
		return UnwrapE2EMessage(vo2Ext.GetMessage())
	}
	if docCap := msg.GetDocumentWithCaptionMessage(); docCap != nil && docCap.GetMessage() != nil {
		return UnwrapE2EMessage(docCap.GetMessage())
	}
	return msg
}

// ExtractContextInfo returns the waE2E.ContextInfo for a raw protobuf message if present.
func ExtractContextInfo(raw []byte) *waE2E.ContextInfo {
	return ExtractContextInfoFromProto(ParseE2EMessage(raw))
}

// ExtractContextInfoFromProto returns the waE2E.ContextInfo for a waE2E.Message if present.
func ExtractContextInfoFromProto(waMsg *waE2E.Message) *waE2E.ContextInfo {
	if waMsg == nil {
		return nil
	}
	switch {
	case waMsg.GetExtendedTextMessage() != nil:
		return waMsg.GetExtendedTextMessage().GetContextInfo()
	case waMsg.GetImageMessage() != nil:
		return waMsg.GetImageMessage().GetContextInfo()
	case waMsg.GetVideoMessage() != nil:
		return waMsg.GetVideoMessage().GetContextInfo()
	case waMsg.GetAudioMessage() != nil:
		return waMsg.GetAudioMessage().GetContextInfo()
	case waMsg.GetDocumentMessage() != nil:
		return waMsg.GetDocumentMessage().GetContextInfo()
	case waMsg.GetStickerMessage() != nil:
		return waMsg.GetStickerMessage().GetContextInfo()
	case waMsg.GetLocationMessage() != nil:
		return waMsg.GetLocationMessage().GetContextInfo()
	case waMsg.GetLiveLocationMessage() != nil:
		return waMsg.GetLiveLocationMessage().GetContextInfo()
	case waMsg.GetContactMessage() != nil:
		return waMsg.GetContactMessage().GetContextInfo()
	case waMsg.GetPollCreationMessage() != nil:
		return waMsg.GetPollCreationMessage().GetContextInfo()
	case waMsg.GetPollCreationMessageV2() != nil:
		return waMsg.GetPollCreationMessageV2().GetContextInfo()
	case waMsg.GetPollCreationMessageV3() != nil:
		return waMsg.GetPollCreationMessageV3().GetContextInfo()
	case waMsg.GetTemplateMessage() != nil:
		return waMsg.GetTemplateMessage().GetContextInfo()
	case waMsg.GetInteractiveMessage() != nil:
		return waMsg.GetInteractiveMessage().GetContextInfo()
	case waMsg.GetButtonsMessage() != nil:
		return waMsg.GetButtonsMessage().GetContextInfo()
	case waMsg.GetListMessage() != nil:
		return waMsg.GetListMessage().GetContextInfo()
	case waMsg.GetOrderMessage() != nil:
		return waMsg.GetOrderMessage().GetContextInfo()
	}
	return nil
}

// ExtractMentionedJIDs returns any JIDs explicitly mentioned in a message's context.
func ExtractMentionedJIDs(raw []byte) []string {
	ctxInfo := ExtractContextInfo(raw)
	if ctxInfo != nil {
		return ctxInfo.GetMentionedJID()
	}
	return nil
}
