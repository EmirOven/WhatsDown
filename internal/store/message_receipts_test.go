package store_test

import (
	"testing"
	"time"

	"postbetasoftware.com/whatsdown/internal/store"
)

func TestGroupReceiptsRequireEveryRecipient(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	chat, id := "123@g.us", "group-message"
	if err := s.SaveMessage(ctx, store.StoredMessage{ChatJID: chat, ID: id, FromMe: true, Kind: store.MessageKindText, Status: store.MessageStatusSent}); err != nil {
		t.Fatal(err)
	}
	recipients := []string{"1@s.whatsapp.net", "2@s.whatsapp.net", "3@s.whatsapp.net"}
	if err := s.SetMessageRecipients(ctx, chat, id, recipients); err != nil {
		t.Fatal(err)
	}
	assertStatus := func(want int) {
		t.Helper()
		msg, err := s.GetMessage(ctx, chat, id)
		if err != nil || msg == nil || msg.Status != want {
			t.Fatalf("message status = %+v, %v; want %d", msg, err, want)
		}
	}
	receipt := func(recipient string, status int) {
		t.Helper()
		if err := s.SaveMessageReceipt(ctx, chat, []string{id}, recipient, status, time.Unix(100, 0)); err != nil {
			t.Fatal(err)
		}
	}
	receipt(recipients[0], store.MessageStatusRead)
	assertStatus(store.MessageStatusSent)
	receipt(recipients[1], store.MessageStatusDelivered)
	assertStatus(store.MessageStatusSent)
	receipt("outsider@s.whatsapp.net", store.MessageStatusRead)
	assertStatus(store.MessageStatusSent)
	receipt(recipients[2], store.MessageStatusDelivered)
	assertStatus(store.MessageStatusDelivered)
	receipt(recipients[1], store.MessageStatusRead)
	assertStatus(store.MessageStatusDelivered)
	receipt(recipients[2], store.MessageStatusRead)
	assertStatus(store.MessageStatusRead)
	receipt(recipients[0], store.MessageStatusDelivered)
	assertStatus(store.MessageStatusRead)
	// A changed membership list must not change recipients of an already sent message.
	if err := s.SetMessageRecipients(ctx, chat, id, []string{"new@s.whatsapp.net"}); err != nil {
		t.Fatal(err)
	}
	assertStatus(store.MessageStatusRead)
	info, known, err := s.GetMessageReceipts(ctx, chat, id)
	if err != nil || !known || len(info) != 4 {
		t.Fatalf("receipt info = %+v, %v, %v", info, known, err)
	}
	for _, r := range info {
		if r.RecipientJID == recipients[0] && (r.ReadAt.IsZero() || r.DeliveredAt.IsZero()) {
			t.Fatalf("receipt timestamps missing: %+v", r)
		}
	}
}

func TestReceiptBeforeRecipientSnapshotAndChatIsolation(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	for _, chat := range []string{"a@g.us", "b@g.us"} {
		if err := s.SaveMessage(ctx, store.StoredMessage{ChatJID: chat, ID: "same", FromMe: true, Kind: store.MessageKindText, Status: store.MessageStatusSent}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SaveMessageReceipt(ctx, "a@g.us", []string{"same"}, "1@s.whatsapp.net", store.MessageStatusRead, time.Now()); err != nil {
		t.Fatal(err)
	}
	msg, _ := s.GetMessage(ctx, "a@g.us", "same")
	if msg.Status != store.MessageStatusSent {
		t.Fatal("unknown recipient list promoted group status")
	}
	if err := s.SetMessageRecipients(ctx, "a@g.us", "same", []string{"1@s.whatsapp.net"}); err != nil {
		t.Fatal(err)
	}
	msg, _ = s.GetMessage(ctx, "a@g.us", "same")
	if msg.Status != store.MessageStatusRead {
		t.Fatal("snapshot lost earlier receipt")
	}
	other, _ := s.GetMessage(ctx, "b@g.us", "same")
	if other.Status != store.MessageStatusSent {
		t.Fatal("receipt leaked to another chat")
	}
}

func TestDirectReceiptPreservesReadWhenDeliveryArrivesLater(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	chat := "1@s.whatsapp.net"
	if err := s.SaveMessage(ctx, store.StoredMessage{ChatJID: chat, ID: "direct", FromMe: true, Status: store.MessageStatusSent}); err != nil {
		t.Fatal(err)
	}
	for _, status := range []int{store.MessageStatusRead, store.MessageStatusDelivered} {
		if err := s.SaveMessageReceipt(ctx, chat, []string{"direct"}, chat, status, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	msg, _ := s.GetMessage(ctx, chat, "direct")
	if msg.Status != store.MessageStatusRead {
		t.Fatalf("direct status = %d", msg.Status)
	}
}
