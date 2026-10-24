package app

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestLiveInactiveReceiptUpdatesRecipient(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	s, err := store.OpenMessageStore(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	previous := msgStore
	msgStore = s
	defer func() { msgStore = previous }()
	chat, id := types.NewJID("123", types.GroupServer), "inactive-receipt"
	if err := s.SaveMessage(ctx, store.StoredMessage{ChatJID: chat.String(), ID: id, FromMe: true, Kind: store.MessageKindText}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMessageRecipients(ctx, chat.String(), id, []string{"1@s.whatsapp.net", "2@s.whatsapp.net"}); err != nil {
		t.Fatal(err)
	}
	handleReceiptEvent(&events.Receipt{
		MessageSource: types.MessageSource{Chat: chat, IsGroup: true, Sender: types.NewJID("1", types.DefaultUserServer)},
		Type:          types.ReceiptTypeInactive, MessageIDs: []types.MessageID{id}, Timestamp: time.Unix(200, 0),
	})
	deadline := time.Now().Add(time.Second)
	for {
		receipts, _, err := s.GetMessageReceipts(ctx, chat.String(), id)
		if err != nil {
			t.Fatal(err)
		}
		if len(receipts) == 2 && receipts[0].RecipientJID == "1@s.whatsapp.net" && receipts[0].Status == store.MessageStatusDelivered {
			if receipts[0].DeliveredAt.Unix() != 200 || receipts[1].Status != store.MessageStatusSent {
				t.Fatalf("incorrect live receipts: %+v", receipts)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("inactive delivery event was lost: %+v", receipts)
		}
		time.Sleep(time.Millisecond)
	}
}
