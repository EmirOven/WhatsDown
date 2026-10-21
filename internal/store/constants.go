package store

// Delivery / read status of outgoing messages.
const (
	MessageStatusNone      = 0 // pending / unacknowledged
	MessageStatusSent      = 1 // server ack (single check)
	MessageStatusDelivered = 2 // delivered to recipient (double check)
	MessageStatusRead      = 3 // read / viewed by recipient (blue double check)
)

// Call outcome constants.
const (
	CallOutcomeConnected = 0
	CallOutcomeMissed    = 1
	CallOutcomeFailed    = 2
	CallOutcomeRejected  = 3
	CallOutcomeDeclined  = 4
)

// Values of whatsdown_messages.kind.
const (
	MessageKindText     = "text"
	MessageKindImage    = "image"
	MessageKindVideo    = "video"
	MessageKindAudio    = "audio"
	MessageKindVoice    = "voice"
	MessageKindDocument = "document"
	MessageKindSticker  = "sticker"
	MessageKindLocation = "location"
	MessageKindContact  = "contact"
	MessageKindPoll     = "poll"
	MessageKindCall     = "call"
	MessageKindDeleted  = "deleted"
)
