package store_test

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestCleanupDuplicateChats(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()

	pnJID := types.NewJID("1234567890", types.DefaultUserServer)
	lidJID := types.NewJID("9988776655", "lid")

	// Setup lid map
	_, err := db.ExecContext(ctx, `INSERT INTO whatsmeow_lid_map (lid, pn) VALUES ('9988776655', '1234567890')`)
	if err != nil {
		t.Fatalf("insert lid map: %v", err)
	}

	// Save message to PN chat
	msg1 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: pnJID, Sender: pnJID, IsFromMe: false},
			ID:            "msg-pn",
			PushName:      "Alice",
			Timestamp:     time.Now().Add(-10 * time.Minute),
		},
		Message: &waE2E.Message{Conversation: proto.String("Hi from PN")},
	}
	if err := s.SaveLiveMessage(ctx, msg1); err != nil {
		t.Fatalf("save live msg1: %v", err)
	}

	// Manually insert an old LID chat with unread count
	_, err = db.ExecContext(ctx, `
		INSERT INTO whatsdown_chats (jid, name, last_message_at, unread_count, pinned)
		VALUES (?, 'Alice', ?, 2, 1)`,
		lidJID.String(), time.Now().Unix())
	if err != nil {
		t.Fatalf("insert lid chat: %v", err)
	}

	// Set pinned using lid JID should update both or resolved PN
	if err := s.SetChatPinned(ctx, lidJID.String(), true); err != nil {
		t.Fatalf("set pinned: %v", err)
	}

	// Run cleanup
	s.CleanupDuplicateChats(ctx)

	chats, err := s.ListChats(ctx)
	if err != nil {
		t.Fatalf("list chats: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("expected 1 deduplicated chat, got %d", len(chats))
	}
	if chats[0].JID != pnJID.String() {
		t.Errorf("expected canonical chat JID %s, got %s", pnJID.String(), chats[0].JID)
	}
	if !chats[0].Pinned {
		t.Errorf("expected merged chat to be pinned")
	}
}
