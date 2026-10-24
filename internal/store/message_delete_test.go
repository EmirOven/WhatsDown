package store_test

import (
	"testing"
	"time"

	"postbetasoftware.com/whatsdown/internal/store"
)

func TestDeleteMessageLocalAndEveryone(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	for _, chat := range []string{"a@s.whatsapp.net", "b@s.whatsapp.net"} {
		if err := s.SaveMessage(ctx, store.StoredMessage{ChatJID: chat, ID: "same-id", Text: "hello", Kind: store.MessageKindText, Timestamp: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DeleteMessage(ctx, "a@s.whatsapp.net", "same-id", true); err != nil {
		t.Fatal(err)
	}
	msg, err := s.GetMessage(ctx, "a@s.whatsapp.net", "same-id")
	if err != nil || msg == nil || msg.Kind != store.MessageKindDeleted || msg.Text != "" || len(msg.Raw) != 0 {
		t.Fatalf("revoked message = %+v, %v", msg, err)
	}
	other, err := s.GetMessage(ctx, "b@s.whatsapp.net", "same-id")
	if err != nil || other == nil || other.Text != "hello" {
		t.Fatalf("other chat changed: %+v, %v", other, err)
	}
	if err := s.DeleteMessage(ctx, "a@s.whatsapp.net", "same-id", false); err != nil {
		t.Fatal(err)
	}
	messages, err := s.ListMessages(ctx, "a@s.whatsapp.net", time.Time{}, 100)
	if err != nil || len(messages) != 0 {
		t.Fatalf("local deletion left messages: %+v, %v", messages, err)
	}
}
