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

func TestMessageQueriesIncludeReactions(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, chat := range []string{"first@g.us", "second@g.us"} {
		if err := s.SaveMessage(ctx, store.StoredMessage{ChatJID: chat, ID: "shared-id", Kind: store.MessageKindText, Text: "hello", Timestamp: time.Now()}); err != nil {
			t.Fatal(err)
		}
		if err := s.SaveReaction(ctx, store.Reaction{ChatJID: chat, MessageID: "shared-id", SenderJID: "sender@s.whatsapp.net", Emoji: chat, Timestamp: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	check := func(msg *store.StoredMessage, err error) {
		t.Helper()
		if err != nil || msg == nil || len(msg.Reactions) != 1 || msg.Reactions[0].Emoji != msg.ChatJID {
			t.Fatalf("message reactions missing or mixed across chats: %+v, %v", msg, err)
		}
	}
	check(s.GetMessage(ctx, "first@g.us", "shared-id"))
	check(s.GetMessageByID(ctx, "shared-id"))
	messages, err := s.ListMessages(ctx, "first@g.us", time.Time{}, 100)
	if err != nil || len(messages) != 1 {
		t.Fatalf("list messages: %v, %v", messages, err)
	}
	check(&messages[0], nil)
	if err := s.RemoveReaction(ctx, "first@g.us", "shared-id", "sender@s.whatsapp.net"); err != nil {
		t.Fatal(err)
	}
	msg, err := s.GetMessage(ctx, "first@g.us", "shared-id")
	if err != nil || msg == nil || len(msg.Reactions) != 0 {
		t.Fatalf("removed reaction still present: %+v, %v", msg, err)
	}
}
