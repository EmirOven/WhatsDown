package store_test

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
)

func syncedReceiptMessage() *waWeb.WebMessageInfo {
	return &waWeb.WebMessageInfo{
		Key:                             &waCommon.MessageKey{RemoteJID: proto.String("123@g.us"), FromMe: proto.Bool(true), ID: proto.String("sync-receipts")},
		OriginalSelfAuthorUserJIDString: proto.String("me@s.whatsapp.net"),
		MessageTimestamp:                proto.Uint64(100),
		Message:                         &waE2E.Message{Conversation: proto.String("hello")},
		UserReceipt: []*waWeb.UserReceipt{
			{UserJID: proto.String("1@s.whatsapp.net"), ReadTimestamp: proto.Int64(200)},
			{UserJID: proto.String("2@s.whatsapp.net"), ReadTimestamp: proto.Int64(201)},
			{UserJID: proto.String("3@s.whatsapp.net"), ReceiptTimestamp: proto.Int64(150)},
			{UserJID: proto.String("4@s.whatsapp.net")},
			{UserJID: proto.String("5@s.whatsapp.net")},
		},
	}
}

func TestPhoneMetadataPreservesAllRecipients(t *testing.T) {
	for _, history := range []bool{false, true} {
		name := "phone response"
		if history {
			name = "history sync"
		}
		t.Run(name, func(t *testing.T) {
			s, ctx, db := setupTestStore(t)
			defer db.Close()
			web := syncedReceiptMessage()
			chat, id := web.GetKey().GetRemoteJID(), web.GetKey().GetID()
			if history {
				_, err := s.SaveHistorySync(ctx, &whatsmeow.Client{}, &waHistorySync.HistorySync{Conversations: []*waHistorySync.Conversation{{
					ID: proto.String(chat), Name: proto.String("Group"), Messages: []*waHistorySync.HistorySyncMsg{{Message: web}},
				}}})
				if err != nil {
					t.Fatal(err)
				}
			} else {
				evt, err := (&whatsmeow.Client{}).ParseWebMessage(types.EmptyJID, web)
				if err != nil {
					t.Fatal(err)
				}
				if err := s.SaveLiveMessage(ctx, evt); err != nil {
					t.Fatal(err)
				}
			}
			receipts, known, err := s.GetMessageReceipts(ctx, chat, id)
			if err != nil || len(receipts) != 5 {
				t.Fatalf("receipts = %+v, %v; want all five recipients", receipts, err)
			}
			if known {
				t.Fatal("partial receipt metadata claimed a complete send roster")
			}
			counts := map[int]int{}
			for _, r := range receipts {
				counts[r.Status]++
			}
			if counts[store.MessageStatusRead] != 2 || counts[store.MessageStatusDelivered] != 1 || counts[store.MessageStatusSent] != 2 {
				t.Fatalf("incorrect receipt counts: %v", counts)
			}
			// Updated metadata for an already stored message must still be imported.
			web.UserReceipt[3].ReadTimestamp = proto.Int64(250)
			evt, err := (&whatsmeow.Client{}).ParseWebMessage(types.EmptyJID, web)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.SaveLiveMessage(ctx, evt); err != nil {
				t.Fatal(err)
			}
			receipts, _, err = s.GetMessageReceipts(ctx, chat, id)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range receipts {
				if r.RecipientJID == "4@s.whatsapp.net" && (r.Status != store.MessageStatusRead || r.ReadAt.Unix() != 250 || !r.DeliveredAt.IsZero()) {
					t.Fatalf("phone read receipt missing or delivery time fabricated: %+v", r)
				}
			}
			// Metadata-only phone updates also work without reinserting a message body.
			web.UserReceipt[2].ReadTimestamp = proto.Int64(260)
			evt.Message = nil
			if err := s.SaveLiveMessage(ctx, evt); err != nil {
				t.Fatal(err)
			}
			receipts, _, _ = s.GetMessageReceipts(ctx, chat, id)
			for _, r := range receipts {
				if r.RecipientJID == "3@s.whatsapp.net" && r.Status != store.MessageStatusRead {
					t.Fatal("metadata-only update lost")
				}
			}
			// A reply alone supplies no receipt for recipient 5.
			if err := s.SaveLiveMessage(ctx, &events.Message{Info: types.MessageInfo{
				MessageSource: types.MessageSource{Chat: types.NewJID("123", types.GroupServer), Sender: types.NewJID("5", types.DefaultUserServer)},
				ID:            "reply", Timestamp: time.Unix(300, 0),
			}, Message: &waE2E.Message{Conversation: proto.String("reply")}}); err != nil {
				t.Fatal(err)
			}
			receipts, _, _ = s.GetMessageReceipts(ctx, chat, id)
			for _, r := range receipts {
				if r.RecipientJID == "5@s.whatsapp.net" && r.Status != store.MessageStatusSent {
					t.Fatal("reply was treated as a read receipt")
				}
			}
		})
	}
}

func TestReceiptIdentityMappingArrivesAfterReceipt(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	db.SetMaxOpenConns(1)
	chat, id := "123@g.us", "identity"
	if err := s.SaveMessage(ctx, store.StoredMessage{ChatJID: chat, ID: id, FromMe: true, Kind: store.MessageKindText}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMessageRecipients(ctx, chat, id, []string{"1@s.whatsapp.net", "2@s.whatsapp.net"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMessageReceipt(ctx, chat, []string{id}, "999:3@lid", store.MessageStatusRead, time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO whatsmeow_lid_map (lid, pn) VALUES ('999', '1')`); err != nil {
		t.Fatal(err)
	}
	receipts, known, err := s.GetMessageReceipts(ctx, chat, id)
	if err != nil || !known || len(receipts) != 2 {
		t.Fatalf("duplicate recipient identities: %+v, %v, %v", receipts, known, err)
	}
	for _, r := range receipts {
		if r.RecipientJID == "1@s.whatsapp.net" && (!r.Expected || r.Status != store.MessageStatusRead || r.ReadAt.Unix() != 200) {
			t.Fatalf("lost mapped receipt: %+v", r)
		}
	}
}

func TestGroupReceiptsExcludeAuthorAndOwnIdentities(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	if _, err := db.ExecContext(ctx, `CREATE TABLE whatsmeow_device (jid TEXT, lid TEXT);
 INSERT INTO whatsmeow_device VALUES ('123:2@s.whatsapp.net', '999@lid');
 INSERT INTO whatsmeow_lid_map (pn,lid) VALUES ('123','999');`); err != nil {
		t.Fatal(err)
	}
	chat, id := "group@g.us", "self-recipient"
	if err := s.SaveMessage(ctx, store.StoredMessage{ChatJID: chat, ID: id, SenderJID: "123@s.whatsapp.net", FromMe: true, Kind: store.MessageKindText, Status: store.MessageStatusSent}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMessageRecipients(ctx, chat, id, []string{"123@s.whatsapp.net", "999:4@lid", "456@s.whatsapp.net"}); err != nil {
		t.Fatal(err)
	}
	// Repair an existing bogus pending self row, too.
	if _, err := db.ExecContext(ctx, `INSERT OR REPLACE INTO whatsdown_message_recipients (chat_jid,message_id,recipient_jid,expected) VALUES (?,?,'999@lid',1)`, chat, id); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMessageReceipt(ctx, chat, []string{id}, "456@s.whatsapp.net", store.MessageStatusRead, time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	receipts, known, err := s.GetMessageReceipts(ctx, chat, id)
	if err != nil || !known || len(receipts) != 1 || receipts[0].RecipientJID != "456@s.whatsapp.net" {
		t.Fatalf("self counted as recipient: %+v, %v, %v", receipts, known, err)
	}
	msg, err := s.GetMessage(ctx, chat, id)
	if err != nil || msg.Status != store.MessageStatusRead {
		t.Fatalf("self blocked group read status: %+v, %v", msg, err)
	}
}

func TestWhatsAppAggregateReadSurvivesIncompleteRoster(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	web := syncedReceiptMessage()
	web.Status = waWeb.WebMessageInfo_READ.Enum()
	evt, err := (&whatsmeow.Client{}).ParseWebMessage(types.EmptyJID, web)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveLiveMessage(ctx, evt); err != nil {
		t.Fatal(err)
	}
	receipts, known, err := s.GetMessageReceipts(ctx, evt.Info.Chat.String(), evt.Info.ID)
	if err != nil || known {
		t.Fatalf("receipt list = %+v, %v, %v", receipts, known, err)
	}
	msg, err := s.GetMessage(ctx, evt.Info.Chat.String(), evt.Info.ID)
	if err != nil || msg.Status != store.MessageStatusRead {
		t.Fatalf("WhatsApp aggregate read status discarded: %+v, %v", msg, err)
	}
	pending := 0
	for _, r := range receipts {
		if r.Status == store.MessageStatusSent {
			pending++
		}
	}
	if pending != 2 {
		t.Fatal("aggregate status fabricated individual read receipts")
	}
}
