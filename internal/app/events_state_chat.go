package app

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/avatar"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func handleGroupInfo() {
	if waClient != nil && msgStore != nil {
		go func() {
			groups, err := waClient.GetJoinedGroups(context.Background())
			if err == nil {
				_ = msgStore.SyncGroups(context.Background(), groups)
				glib.IdleAdd(func() {
					if a := ui.ActiveMainWindow(); a != nil {
						a.LoadChats()
					}
				})
			}
		}()
	}
}

func handleChatStateAction(evt any) {
	if msgStore == nil {
		return
	}
	// Applying on the dispatcher provides backpressure during state replay.
	ctx := context.Background()
	switch e := evt.(type) {
	case *events.Pin:
		if e.Action != nil {
			_ = msgStore.SetChatPinned(ctx, e.JID.ToNonAD().String(), e.Action.GetPinned())
			refreshChatsUI()
		}
	case *events.Archive:
		if e.Action != nil {
			_ = msgStore.SetChatArchived(ctx, e.JID.ToNonAD().String(), e.Action.GetArchived())
			refreshChatsUI()
		}
	case *events.Mute:
		if e.Action != nil {
			_ = msgStore.SetChatMuted(ctx, e.JID.ToNonAD().String(), e.Action.GetMuted())
			refreshChatsUI()
		}
	case *events.Star:
		if e.Action != nil {
			_ = msgStore.ToggleMessageStarred(ctx, e.ChatJID.ToNonAD().String(), e.MessageID, e.Action.GetStarred())
			messageRefreshNeeded.Store(true)
			refreshChatsUI()
		}
	case *events.MarkChatAsRead:
		if e.Action != nil && e.Action.GetRead() {
			ui.WithdrawChatNotification(e.JID.ToNonAD().String())
			at := e.Timestamp
			if ts := e.Action.GetMessageRange().GetLastMessageTimestamp(); ts > 0 {
				rangeAt := time.Unix(ts, 0)
				if at.IsZero() || rangeAt.After(at) {
					at = rangeAt
				}
			}
			if at.IsZero() {
				at = time.Now()
			}
			if err := msgStore.MarkChatReadAt(ctx, e.JID, at); err != nil {
				log.Printf("syncing chat read state: %v", err)
			}
			refreshChatsUI()
		}
	}
}

func handlePictureUpdate(e *events.Picture) {
	jidStr := e.JID.ToNonAD().String()
	avatar.Invalidate(jidStr)
	glib.IdleAdd(func() {
		if a := ui.ActiveMainWindow(); a != nil {
			a.LoadChats()
			a.RefreshActiveChatAvatar(jidStr)
		}
	})
}
