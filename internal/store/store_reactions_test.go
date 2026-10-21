package store_test

import (
	"testing"
	"time"

	"postbetasoftware.com/whatsdown/internal/store"
)

func TestReactionsStore(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()

	chatJID := "chat-123@s.whatsapp.net"
	msgID := "msg-react-1"

	r1 := store.Reaction{
		ChatJID:    chatJID,
		MessageID:  msgID,
		SenderJID:  "sender1@s.whatsapp.net",
		SenderName: "Sender 1",
		Emoji:      "❤️",
		FromMe:     false,
		Timestamp:  time.Now(),
	}
	if err := s.SaveReaction(ctx, r1); err != nil {
		t.Fatalf("saveReaction: %v", err)
	}

	reactions, err := s.GetReactionsForMessage(ctx, chatJID, msgID)
	if err != nil || len(reactions) != 1 || reactions[0].Emoji != "❤️" {
		t.Fatalf("unexpected reactions: %+v (err: %v)", reactions, err)
	}

	allReactions, err := s.GetReactionsForChat(ctx, chatJID)
	if err != nil || len(allReactions[msgID]) != 1 {
		t.Fatalf("unexpected all reactions: %+v", allReactions)
	}

	if err := s.RemoveReaction(ctx, chatJID, msgID, "sender1@s.whatsapp.net"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	reactions, _ = s.GetReactionsForMessage(ctx, chatJID, msgID)
	if len(reactions) != 0 {
		t.Errorf("expected 0 reactions after remove")
	}
}
