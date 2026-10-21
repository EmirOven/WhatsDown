package app

import (
	"context"
	"log"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func handleSyncEvent(evt any) {
	switch e := evt.(type) {
	case *events.PairSuccess:
		glib.IdleAdd(func() {
			syncInfo = newSyncState(false)
			if currentPage != pageSyncing {
				if err := showPage(pageSyncing); err != nil {
					log.Printf("showing syncing page: %v", err)
				}
			}
		})

	case *events.HistorySync:
		conversations := e.Data.GetConversations()
		messages := 0
		for _, conv := range conversations {
			messages += len(conv.GetMessages())
		}
		percent := -1
		if e.Data.Progress != nil {
			percent = int(e.Data.GetProgress())
		}
		glib.IdleAdd(func() {
			handleHistorySync(len(conversations), messages, percent)
		})
		go func() {
			if msgStore != nil && waClient != nil {
				stored, err := msgStore.SaveHistorySync(context.Background(), waClient, e.Data)
				if err != nil {
					log.Printf("saving history sync: %v", err)
				} else {
					log.Printf("saved %d messages from history sync", stored)
				}
				glib.IdleAdd(func() {
					if active := ui.ActiveMainWindow(); active != nil {
						active.LoadChats()
						active.LoadArchivedChats()
						active.LoadSavedStickersUI()
					}
				})
			}
		}()

	case *events.OfflineSyncPreview:
		total := e.Messages
		glib.IdleAdd(func() {
			syncInfo.offlineTotal = total
			renderSync()
		})

	case *events.OfflineSyncCompleted:
		glib.IdleAdd(func() {
			if syncInfo.reconnecting {
				syncInfo.done = true
				renderSync()
			}
		})
	}
}
