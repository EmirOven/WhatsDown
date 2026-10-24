package store_test

import (
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
	"testing"
	"time"
)

func TestPhoneReadSurvivesLateMessages(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	chat := types.NewJID("123", types.GroupServer)
	save := func(id string, ts int64) {
		t.Helper()
		if err := s.SaveLiveMessage(ctx, &events.Message{
			Info:    types.MessageInfo{MessageSource: types.MessageSource{Chat: chat, Sender: types.NewJID("456", types.DefaultUserServer)}, ID: id, PushName: "Alice", Timestamp: time.Unix(ts, 0)},
			Message: &waE2E.Message{Conversation: proto.String(id)},
		}); err != nil {
			t.Fatal(err)
		}
	}
	check := func(want int) {
		t.Helper()
		chats, err := s.ListChats(ctx)
		if err != nil || len(chats) != 1 || chats[0].UnreadCount != want {
			t.Fatalf("chats = %+v, %v; want unread %d", chats, err, want)
		}
		if chats[0].LastMessageSender != "Alice" {
			t.Fatalf("missing preview sender: %+v", chats[0])
		}
	}
	// Read can arrive before the messages it covers, including offline replay.
	if err := s.MarkChatReadAt(ctx, chat, time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	save("old", 100)
	check(0)
	save("new", 300)
	check(1)
	if err := s.MarkChatReadAt(ctx, chat, time.Unix(150, 0)); err != nil {
		t.Fatal(err)
	}
	check(1)
	save("late", 180)
	check(1)
	if err := s.MarkChatReadAt(ctx, chat, time.Unix(300, 0)); err != nil {
		t.Fatal(err)
	}
	check(0)
	// Watermarks persist across reopening the store.
	if _, err := store.OpenMessageStore(ctx, db); err != nil {
		t.Fatal(err)
	}
	save("late-again", 250)
	check(0)
}

func TestPhoneReadMatchesBothChatIdentities(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	if _, err := db.ExecContext(ctx, `INSERT INTO whatsmeow_lid_map (lid, pn) VALUES ('999', '123')`); err != nil {
		t.Fatal(err)
	}
	for _, jid := range []string{"123@s.whatsapp.net", "999@lid", "123@g.us"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO whatsdown_chats (jid, unread_count) VALUES (?, 3)`, jid); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.MarkChatReadAt(ctx, types.NewJID("999", types.HiddenUserServer), time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	chats, err := s.ListChats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range chats {
		want := 0
		if ch.JID == "123@g.us" {
			want = 3
		}
		if ch.UnreadCount != want {
			t.Fatalf("unread for %s = %d, want %d", ch.JID, ch.UnreadCount, want)
		}
	}
}

func TestPhoneReadSurvivesHistoryReplay(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()
	jid := types.NewJID("123", types.GroupServer)
	replay := func(ts uint64, unread uint32) {
		t.Helper()
		_, err := s.SaveHistorySync(ctx, nil, &waHistorySync.HistorySync{Conversations: []*waHistorySync.Conversation{{
			ID: proto.String(jid.String()), Name: proto.String("Group"), ConversationTimestamp: proto.Uint64(ts), UnreadCount: proto.Uint32(unread),
		}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	check := func(want int) {
		t.Helper()
		chats, err := s.ListChats(ctx)
		if err != nil || len(chats) != 1 || chats[0].UnreadCount != want {
			t.Fatalf("history chats = %+v, %v; want unread %d", chats, err, want)
		}
	}
	replay(100, 3)
	check(3)
	if err := s.MarkChatReadAt(ctx, jid, time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	replay(100, 3)
	check(0)
	replay(300, 2)
	check(2)
	replay(100, 3)
	check(2)
}
