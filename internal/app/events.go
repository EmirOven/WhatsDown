package app

import (
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func handleWhatsAppEvent(evt any) {
	switch e := evt.(type) {
	case *events.PairSuccess, *events.HistorySync, *events.OfflineSyncPreview, *events.OfflineSyncCompleted:
		handleSyncEvent(e)
	case *events.Message:
		handleMessageEvent(e)
	case *events.Receipt:
		handleReceiptEvent(e)
	case *events.Presence:
		ui.ActiveMainWindow().HandlePresence(e)
	case *events.ChatPresence:
		ui.ActiveMainWindow().HandleChatPresence(e)
	case *events.CallOffer, *events.CallTerminate:
		handleCallEvent(e)
	case *events.Connected, *events.LoggedOut, *events.GroupInfo, *events.Pin, *events.Archive, *events.Mute, *events.Star, *events.AppStateSyncComplete, *events.Picture, *events.Contact, *events.PushName, *events.BusinessName:
		handleStateEvent(e)
	}
}
