package app

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func handleStateEvent(evt any) {
	switch e := evt.(type) {
	case *events.Connected:
		handleConnected()
	case *events.LoggedOut:
		handleLoggedOut(e)
	case *events.GroupInfo:
		handleGroupInfo()
	case *events.Pin, *events.Archive, *events.Mute, *events.Star, *events.MarkChatAsRead:
		handleChatStateAction(e)
	case *events.AppStateSyncComplete:
		refreshChatsUI()
	case *events.Picture:
		handlePictureUpdate(e)
	case *events.Contact, *events.PushName, *events.BusinessName:
		refreshChatsUI()
	}
}

func handleConnected() {
	go func() {
		time.Sleep(500 * time.Millisecond)
		if waClient != nil {
			_ = waClient.SendPresence(context.Background(), types.PresenceAvailable)
		}
		for _, name := range appstate.AllPatchNames {
			if waClient != nil {
				_ = waClient.FetchAppState(context.Background(), name, false, false)
			}
		}
		if msgStore != nil && waClient != nil {
			msgStore.ImportStickersFromChatHistory(context.Background(), waClient)
			groups, err := waClient.GetJoinedGroups(context.Background())
			if err == nil && len(groups) > 0 {
				_ = msgStore.SyncGroups(context.Background(), groups)
				glib.IdleAdd(func() {
					if a := ui.ActiveMainWindow(); a != nil {
						a.LoadChats()
					}
				})
			}
		}
	}()
}

func handleLoggedOut(e *events.LoggedOut) {
	log.Printf("logged out by WhatsApp (reason %v), showing QR login", e.Reason)
	glib.IdleAdd(func() {
		ui.SetActiveMainWindow(nil)
		syncInfo = newSyncState(false)
		qrTexture = nil
		if err := showPage(pageInitial); err != nil {
			log.Printf("showing QR page: %v", err)
		}
		go func() {
			client, err := newWhatsAppClient()
			if err != nil {
				log.Printf("opening WhatsApp session store: %v", err)
				return
			}
			connectWhatsApp(client)
		}()
	})
}
