package store_test

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestChatsAndMute(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()

	chatJID := types.NewJID("987654321", types.DefaultUserServer)
	msg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chatJID, Sender: chatJID, IsFromMe: false},
			ID:            "msg-test-mute",
			PushName:      "Bob",
			Timestamp:     time.Now(),
		},
		Message: &waE2E.Message{Conversation: proto.String("Hi Bob")},
	}
	if err := s.SaveLiveMessage(ctx, msg); err != nil {
		t.Fatalf("save: %v", err)
	}

	chats, _ := s.ListChats(ctx)
	if len(chats) != 1 || chats[0].Muted {
		t.Fatalf("expected unmuted chat")
	}

	if err := s.SetChatMuted(ctx, chatJID.String(), true); err != nil {
		t.Fatalf("set muted: %v", err)
	}
	chats, _ = s.ListChats(ctx)
	if !chats[0].Muted {
		t.Errorf("expected muted chat")
	}

	if err := s.SetChatPinned(ctx, chatJID.String(), true); err != nil {
		t.Fatalf("set pinned: %v", err)
	}
	chats, _ = s.ListChats(ctx)
	if !chats[0].Pinned {
		t.Errorf("expected pinned chat")
	}

	if err := s.SetChatArchived(ctx, chatJID.String(), true); err != nil {
		t.Fatalf("set archived: %v", err)
	}
	archived, _ := s.ListArchivedChats(ctx)
	if len(archived) != 1 {
		t.Errorf("expected 1 archived chat")
	}
}

func TestSeedDemoData(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()

	if err := s.SeedDemoData(ctx); err != nil {
		t.Fatalf("seed demo data: %v", err)
	}
	if !s.IsDemo() {
		t.Errorf("expected store.IsDemo() to be true")
	}
	chats, err := s.ListChats(ctx)
	if err != nil || len(chats) == 0 {
		t.Fatalf("expected seeded demo chats, got %d (err: %v)", len(chats), err)
	}
	archived, _ := s.ListArchivedChats(ctx)
	if len(archived) == 0 {
		t.Errorf("expected seeded archived demo chat")
	}
	calls, _ := s.ListCalls(ctx, "")
	if len(calls) == 0 {
		t.Errorf("expected seeded demo calls")
	}
}
