package app

import (
	"context"
	"log"
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func handleCallEvent(evt any) {
	switch e := evt.(type) {
	case *events.CallOffer:
		log.Printf("incoming call offer: callID=%s from=%s caller=%s", e.CallID, e.From, e.CallCreator)
		if msgStore != nil {
			callerJID := e.CallCreator.ToNonAD().String()
			if callerJID == "" {
				callerJID = e.From.ToNonAD().String()
			}
			chatJID := e.From.ToNonAD().String()
			if chatJID == "" {
				chatJID = callerJID
			}
			go func() {
				_ = msgStore.SaveCall(context.Background(), store.CallRecord{
					ID:        e.CallID,
					ChatJID:   chatJID,
					CallerJID: callerJID,
					Timestamp: e.Timestamp,
					Outcome:   store.CallOutcomeConnected,
					FromMe:    false,
				})
				glib.IdleAdd(func() {
					if active := ui.ActiveMainWindow(); active != nil {
						active.LoadCalls()
					}
				})
			}()
		}

	case *events.CallTerminate:
		log.Printf("call terminate: callID=%s reason=%s", e.CallID, e.Reason)
		if msgStore != nil {
			outcome := store.CallOutcomeConnected
			rLower := strings.ToLower(e.Reason)
			if strings.Contains(rLower, "miss") || strings.Contains(rLower, "timeout") {
				outcome = store.CallOutcomeMissed
			} else if strings.Contains(rLower, "reject") || strings.Contains(rLower, "decline") {
				outcome = store.CallOutcomeRejected
			}
			callerJID := e.CallCreator.ToNonAD().String()
			if callerJID == "" {
				callerJID = e.From.ToNonAD().String()
			}
			chatJID := e.From.ToNonAD().String()
			if chatJID == "" {
				chatJID = callerJID
			}
			go func() {
				_ = msgStore.SaveCall(context.Background(), store.CallRecord{
					ID:        e.CallID,
					ChatJID:   chatJID,
					CallerJID: callerJID,
					Timestamp: e.Timestamp,
					Outcome:   outcome,
					FromMe:    false,
				})
				glib.IdleAdd(func() {
					if active := ui.ActiveMainWindow(); active != nil {
						active.LoadCalls()
					}
				})
			}()
		}
	}
}
