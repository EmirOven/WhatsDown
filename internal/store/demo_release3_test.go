package store_test

import (
	"testing"
	"time"

	"postbetasoftware.com/whatsdown/internal/store"
)

func TestSeedDemoDataRelease3(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()

	if err := s.SeedDemoData(ctx); err != nil {
		t.Fatalf("seed demo data: %v", err)
	}

	folders, err := s.ListChatFolders(ctx)
	if err != nil || len(folders) != 2 {
		t.Fatalf("expected 2 demo folders, got %d (err: %v)", len(folders), err)
	}
	for _, f := range folders {
		if len(f.ChatJIDs) == 0 {
			t.Errorf("folder %q has no chats", f.Name)
		}
	}

	for _, asset := range store.DemoMediaAssets(time.Now()) {
		msg, err := s.GetMessage(ctx, asset.Message.ChatJID, asset.Message.ID)
		if err != nil || msg == nil {
			t.Fatalf("expected seeded media message %s: %v", asset.Message.ID, err)
		}
		if msg.Kind != asset.Message.Kind || len(msg.Raw) == 0 {
			t.Errorf("media message %s: kind %q, raw %d bytes", msg.ID, msg.Kind, len(msg.Raw))
		}
	}

	receipts, known, err := s.GetMessageReceipts(ctx, "120363000000000001@g.us", "demo_r3_core_notes")
	if err != nil || !known || len(receipts) != 4 {
		t.Fatalf("expected 4 known receipts, got %d (known %v, err %v)", len(receipts), known, err)
	}
	read, delivered := 0, 0
	for _, r := range receipts {
		switch r.Status {
		case store.MessageStatusRead:
			read++
			if r.ReadAt.IsZero() || r.DeliveredAt.IsZero() {
				t.Errorf("read receipt for %s is missing timestamps", r.RecipientJID)
			}
		case store.MessageStatusDelivered:
			delivered++
		}
	}
	if read != 2 || delivered != 2 {
		t.Errorf("expected 2 read and 2 delivered receipts, got %d and %d", read, delivered)
	}
	msg, _ := s.GetMessage(ctx, "120363000000000001@g.us", "demo_r3_core_notes")
	if msg == nil || msg.Status != store.MessageStatusDelivered {
		t.Errorf("partially read group message should show delivered ticks, got %+v", msg)
	}

	chats, err := s.ListChats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, chat := range chats {
		if chat.JID == "120363000000000002@g.us" && chat.LastMessageSender == "" {
			t.Errorf("group preview should carry the last sender")
		}
	}
}
