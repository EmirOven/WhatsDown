package app

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func handleIncomingReaction(evt *events.Message, rm *waE2E.ReactionMessage) {
	if msgStore == nil || rm == nil || rm.GetKey() == nil {
		return
	}
	targetID := rm.GetKey().GetID()
	if targetID == "" {
		return
	}
	chatJID := evt.Info.Chat.ToNonAD().String()
	if msgStore.IsSelfJID(context.Background(), chatJID) {
		if canonical := msgStore.GetCanonicalSelfJID(context.Background()); canonical != "" {
			chatJID = canonical
		}
	}
	senderJID := evt.Info.Sender.ToNonAD().String()
	if senderJID == "" {
		senderJID = chatJID
	}
	emoji := rm.GetText()
	senderName := evt.Info.PushName

	if emoji == "" {
		_ = msgStore.RemoveReaction(context.Background(), chatJID, targetID, senderJID)
		if evt.Info.IsFromMe {
			_ = msgStore.RemoveMyReaction(context.Background(), chatJID, targetID)
		}
	} else {
		_ = msgStore.SaveReaction(context.Background(), store.Reaction{
			ChatJID:    chatJID,
			MessageID:  targetID,
			SenderJID:  senderJID,
			SenderName: senderName,
			Emoji:      emoji,
			FromMe:     evt.Info.IsFromMe,
			Timestamp:  evt.Info.Timestamp,
		})
	}

	glib.IdleAdd(func() {
		if a := ui.ActiveMainWindow(); a != nil {
			a.UpdateMessageReactions(chatJID, targetID)
		}
	})
}
