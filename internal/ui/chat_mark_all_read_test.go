package ui

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
)

type readSyncMock struct {
	receipts []struct {
		chat, sender types.JID
		ids          []types.MessageID
	}
	patches []appstate.PatchInfo
	patch   func(appstate.PatchInfo) error
}

func (m *readSyncMock) SendAppState(_ context.Context, p appstate.PatchInfo) error {
	m.patches = append(m.patches, p)
	if m.patch != nil {
		return m.patch(p)
	}
	return nil
}
func (m *readSyncMock) MarkRead(_ context.Context, ids []types.MessageID, _ time.Time, chat, sender types.JID, _ ...types.ReceiptType) error {
	m.receipts = append(m.receipts, struct {
		chat, sender types.JID
		ids          []types.MessageID
	}{chat, sender, append([]types.MessageID(nil), ids...)})
	return nil
}
func saveUnreadTestMessage(t *testing.T, s *store.MessageStore, chat, sender, id string, ts int64) {
	t.Helper()
	cj, _ := types.ParseJID(chat)
	sj, _ := types.ParseJID(sender)
	if err := s.SaveLiveMessage(context.Background(), &events.Message{Info: types.MessageInfo{
		MessageSource: types.MessageSource{Chat: cj, Sender: sj}, ID: id, Timestamp: time.Unix(ts, 0),
	}, Message: &waE2E.Message{Conversation: proto.String(id)}}); err != nil {
		t.Fatal(err)
	}
}
func TestMarkAllReadSyncsWholeChatsAndKeepsNewMessagesUnread(t *testing.T) {
	c := newFolderSidebar(t)
	ctx := context.Background()
	const group = "123@g.us"
	for i := 0; i < 65; i++ {
		saveUnreadTestMessage(t, c.store, group, fmt.Sprintf("%d@s.whatsapp.net", i%2+1), fmt.Sprint(i), int64(100+i))
	}
	direct := "456@s.whatsapp.net"
	saveUnreadTestMessage(t, c.store, direct, direct, "archived", 100)
	if err := c.store.SetChatArchived(ctx, direct, true); err != nil {
		t.Fatal(err)
	}
	mock := &readSyncMock{}
	mock.patch = func(p appstate.PatchInfo) error {
		if p.Mutations[0].Index[1] == group {
			saveUnreadTestMessage(t, c.store, group, "1@s.whatsapp.net", "new-arrival", 300)
		}
		return nil
	}
	if err := markAllUnreadChats(ctx, c.store, mock, time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, call := range mock.receipts {
		if call.chat.String() == group && call.sender.User != "1" && call.sender.User != "2" {
			t.Fatalf("incorrect group receipt sender %v", call.sender)
		}
		for _, id := range call.ids {
			seen[id] = true
		}
	}
	if len(seen) != 66 || seen["new-arrival"] {
		t.Fatalf("wrong receipt targets: %d", len(seen))
	}
	if len(mock.patches) != 2 {
		t.Fatalf("synced %d chats, want active and archived", len(mock.patches))
	}
	chat, err := c.store.GetChatOrContact(ctx, group)
	if err != nil || chat.UnreadCount != 1 {
		t.Fatalf("new arrival was marked read: %+v %v", chat, err)
	}
	chat, err = c.store.GetChatOrContact(ctx, direct)
	if err != nil || chat.UnreadCount != 0 {
		t.Fatalf("archived chat stayed unread: %+v %v", chat, err)
	}
}
func TestMarkAllReadDoesNotHideSyncFailure(t *testing.T) {
	c := newFolderSidebar(t)
	jid := "123@s.whatsapp.net"
	saveUnreadTestMessage(t, c.store, jid, jid, "unread", 100)
	for _, client := range []readSyncClient{nil, &readSyncMock{patch: func(appstate.PatchInfo) error { return errors.New("sync failed") }}} {
		if err := markAllUnreadChats(context.Background(), c.store, client, time.Unix(200, 0)); err == nil {
			t.Fatal("sync failure hidden")
		}
		chat, err := c.store.GetChatOrContact(context.Background(), jid)
		if err != nil || chat.UnreadCount != 1 {
			t.Fatalf("unsynced chat marked read: %+v %v", chat, err)
		}
	}
}
