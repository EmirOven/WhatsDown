package app

import (
	"context"

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
	go func() {
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
				chatJID := e.ChatJID.ToNonAD().String()
				_ = msgStore.ToggleMessageStarred(ctx, chatJID, e.MessageID, e.Action.GetStarred())
				glib.IdleAdd(func() {
					if a := ui.ActiveMainWindow(); a != nil && a.CurrentChat() != nil {
						if a.IsSameChat(a.CurrentChat().JID, chatJID) {
							a.UpdateSingleMessageRow(e.MessageID)
						}
					}
				})
			}
		case *events.MarkChatAsRead:
			if e.Action != nil && e.Action.GetRead() {
				_ = msgStore.MarkChatRead(ctx, e.JID)
				refreshChatsUI()
			}
		}
	}()
}

func refreshChatsUI() {
	glib.IdleAdd(func() {
		if a := ui.ActiveMainWindow(); a != nil {
			a.LoadChats()
			a.LoadArchivedChats()
		}
	})
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
