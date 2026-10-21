package store_test

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestDeleteChatLocally(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()

	chatJID := types.NewJID("5551234", types.DefaultUserServer)
	msg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chatJID, Sender: chatJID, IsFromMe: false},
			ID:            "msg-to-delete",
			PushName:      "DeleteMe",
			Timestamp:     time.Now(),
		},
		Message: &waE2E.Message{Conversation: proto.String("Farewell")},
	}
	if err := s.SaveLiveMessage(ctx, msg); err != nil {
		t.Fatalf("save: %v", err)
	}

	chats, _ := s.ListChats(ctx)
	if len(chats) != 1 {
		t.Fatalf("expected 1 chat before delete")
	}

	if err := s.DeleteChatLocally(ctx, chatJID.String()); err != nil {
		t.Fatalf("delete chat locally: %v", err)
	}

	chats, _ = s.ListChats(ctx)
	if len(chats) != 0 {
		t.Errorf("expected 0 chats after delete, got %d", len(chats))
	}

	msgs, _ := s.ListMessages(ctx, chatJID.String(), time.Time{}, 50)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(msgs))
	}
}
