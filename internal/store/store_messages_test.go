package store_test

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestLiveMessageAndEdit(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()

	chatJID := types.NewJID("123456789", types.DefaultUserServer)
	msg1 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chatJID, Sender: chatJID, IsFromMe: false},
			ID:            "msg-1",
			PushName:      "Alice",
			Timestamp:     time.Unix(1700000000, 0),
		},
		Message: &waE2E.Message{Conversation: proto.String("Hello from Alice")},
	}

	if err := s.SaveLiveMessage(ctx, msg1); err != nil {
		t.Fatalf("saveLiveMessage 1: %v", err)
	}

	chats, err := s.ListChats(ctx)
	if err != nil || len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d (err: %v)", len(chats), err)
	}
	if chats[0].UnreadCount != 1 || chats[0].Name != "Alice" {
		t.Errorf("unexpected chat state: %+v", chats[0])
	}

	if err := s.MarkChatRead(ctx, chatJID); err != nil {
		t.Fatalf("markChatRead: %v", err)
	}
	chats, _ = s.ListChats(ctx)
	if chats[0].UnreadCount != 0 {
		t.Errorf("expected 0 unread, got %d", chats[0].UnreadCount)
	}

	editMsg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chatJID, Sender: chatJID, IsFromMe: false},
			ID:            "msg-edit-1",
			Timestamp:     time.Unix(1700000010, 0),
		},
		Message: &waE2E.Message{
			ProtocolMessage: &waE2E.ProtocolMessage{
				Key:           &waCommon.MessageKey{RemoteJID: proto.String(chatJID.String()), ID: proto.String("msg-1")},
				Type:          waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
				EditedMessage: &waE2E.Message{Conversation: proto.String("Hello from Alice (edited)")},
			},
		},
	}
	if err := s.SaveLiveMessage(ctx, editMsg); err != nil {
		t.Fatalf("saveLiveMessage edit: %v", err)
	}

	msg, err := s.GetMessage(ctx, chatJID.String(), "msg-1")
	if err != nil || msg == nil {
		t.Fatalf("failed to fetch message: %v", err)
	}
	if msg.Text != "Hello from Alice (edited)" || !msg.Edited {
		t.Errorf("unexpected edited msg: %+v", msg)
	}
}
