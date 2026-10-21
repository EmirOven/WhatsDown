package app

import (
	"context"
	"log"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func handleMessageEvent(e *events.Message) {
	go func() {
		if e.Message != nil && waClient != nil && msgStore != nil {
			var rm *waE2E.ReactionMessage
			if e.Message.GetReactionMessage() != nil {
				rm = e.Message.GetReactionMessage()
			} else if e.Message.GetEncReactionMessage() != nil {
				rm, _ = waClient.DecryptReaction(context.Background(), e)
			}
			if rm != nil {
				handleIncomingReaction(e, rm)
				return
			}
		}
		if e.Message != nil && e.Message.GetPollUpdateMessage() != nil && waClient != nil && msgStore != nil {
			handleIncomingPollUpdate(e)
		}
		if msgStore != nil {
			if err := msgStore.SaveLiveMessage(context.Background(), e); err != nil {
				log.Printf("saving live message: %v", err)
			}
			if e.Info.IsFromMe && e.Message != nil && e.Message.GetStickerMessage() != nil {
				go msgStore.ImportStickersFromChatHistory(context.Background(), waClient)
			}
		}
		glib.IdleAdd(func() {
			if syncInfo.reconnecting && !syncInfo.done {
				syncInfo.offlineReceived++
				renderSync()
			}
			if active := ui.ActiveMainWindow(); active != nil {
				active.HandleIncomingLiveMessage(e)
			}
		})
	}()
}
