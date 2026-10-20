package main

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestMessageStore(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	chatJID := types.NewJID("123456789", types.DefaultUserServer)
	senderJID := chatJID

	// 1. Save live text message
	msg1 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   senderJID,
				IsFromMe: false,
			},
			ID:        "msg-1",
			PushName:  "Alice",
			Timestamp: time.Unix(1700000000, 0),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Hello from Alice"),
		},
	}

	if err := store.saveLiveMessage(ctx, msg1); err != nil {
		t.Fatalf("saveLiveMessage 1: %v", err)
	}

	// Verify chat created with unread_count = 1
	chats, err := store.listChats(ctx)
	if err != nil {
		t.Fatalf("listChats: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
	if chats[0].UnreadCount != 1 || chats[0].Name != "Alice" {
		t.Errorf("unexpected chat state: %+v", chats[0])
	}

	// 2. Mark chat read
	if err := store.markChatRead(ctx, chatJID); err != nil {
		t.Fatalf("markChatRead: %v", err)
	}
	chats, _ = store.listChats(ctx)
	if chats[0].UnreadCount != 0 {
		t.Errorf("expected 0 unread, got %d", chats[0].UnreadCount)
	}

	// 3. Edit message
	editMsg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   senderJID,
				IsFromMe: false,
			},
			ID:        "msg-edit-1",
			Timestamp: time.Unix(1700000010, 0),
		},
		Message: &waE2E.Message{
			ProtocolMessage: &waE2E.ProtocolMessage{
				Key: &waCommon.MessageKey{
					RemoteJID: proto.String(chatJID.String()),
					ID:        proto.String("msg-1"),
				},
				Type: waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
				EditedMessage: &waE2E.Message{
					Conversation: proto.String("Hello from Alice (edited)"),
				},
			},
		},
	}
	if err := store.saveLiveMessage(ctx, editMsg); err != nil {
		t.Fatalf("saving edit: %v", err)
	}

	msgs, err := store.listMessages(ctx, chatJID.String(), time.Time{}, 10)
	if err != nil {
		t.Fatalf("listMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Text != "Hello from Alice (edited)" || !msgs[0].Edited {
		t.Errorf("unexpected message state after edit: %+v", msgs[0])
	}

	// 4. Revoke (delete) message
	revokeMsg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   senderJID,
				IsFromMe: false,
			},
			ID:        "msg-revoke-1",
			Timestamp: time.Unix(1700000020, 0),
		},
		Message: &waE2E.Message{
			ProtocolMessage: &waE2E.ProtocolMessage{
				Key: &waCommon.MessageKey{
					RemoteJID: proto.String(chatJID.String()),
					ID:        proto.String("msg-1"),
				},
				Type: waE2E.ProtocolMessage_REVOKE.Enum(),
			},
		},
	}
	if err := store.saveLiveMessage(ctx, revokeMsg); err != nil {
		t.Fatalf("saving revoke: %v", err)
	}

	msgs, _ = store.listMessages(ctx, chatJID.String(), time.Time{}, 10)
	if len(msgs) != 1 || msgs[0].Kind != messageKindDeleted || msgs[0].Text != "" {
		t.Errorf("unexpected message state after revoke: %+v", msgs[0])
	}

	// 5. Test Pinned and Archived
	if err := store.setChatPinned(ctx, chatJID.String(), true); err != nil {
		t.Fatalf("setChatPinned: %v", err)
	}
	chats, err = store.listChats(ctx)
	if err != nil || len(chats) != 1 || !chats[0].Pinned {
		t.Fatalf("expected chat to be pinned, got: %+v, err: %v", chats, err)
	}

	if err := store.setChatArchived(ctx, chatJID.String(), true); err != nil {
		t.Fatalf("setChatArchived: %v", err)
	}
	chats, err = store.listChats(ctx)
	if err != nil || len(chats) != 0 {
		t.Fatalf("expected 0 active chats after archiving, got %d", len(chats))
	}
	archivedChats, err := store.listArchivedChats(ctx)
	if err != nil || len(archivedChats) != 1 || !archivedChats[0].Archived {
		t.Fatalf("expected 1 archived chat, got: %+v, err: %v", archivedChats, err)
	}

	// Unarchive
	if err := store.setChatArchived(ctx, chatJID.String(), false); err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	chats, _ = store.listChats(ctx)
	if len(chats) != 1 {
		t.Fatalf("expected 1 active chat after unarchiving, got %d", len(chats))
	}

	// 6. Test Media Messages
	mediaMsg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   senderJID,
				IsFromMe: false,
			},
			ID:        "media-1",
			PushName:  "Alice",
			Timestamp: time.Unix(1700000030, 0),
		},
		Message: &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption: proto.String("Look at this photo"),
			},
		},
	}
	if err := store.saveLiveMessage(ctx, mediaMsg); err != nil {
		t.Fatalf("saving media message: %v", err)
	}
	mediaList, err := store.listMediaMessages(ctx, chatJID.String())
	if err != nil {
		t.Fatalf("listMediaMessages: %v", err)
	}
	if len(mediaList) != 1 || mediaList[0].Kind != messageKindImage || mediaList[0].Text != "Look at this photo" {
		t.Errorf("unexpected media list: %+v", mediaList)
	}

	// 7. Test getStats
	chatCount, msgCount, err := store.getStats(ctx)
	if err != nil {
		t.Fatalf("getStats: %v", err)
	}
	if chatCount != 1 || msgCount != 2 {
		t.Errorf("expected 1 chat and 2 messages, got %d chats and %d messages", chatCount, msgCount)
	}
}

func TestFormatDisplayNameAndCleanPhone(t *testing.T) {
	// 1. cleanPhoneNumber
	tests := []struct {
		input    string
		expected string
	}{
		{"1234567890@s.whatsapp.net", "+1234567890"},
		{"1234567890:5@s.whatsapp.net", "+1234567890"},
		{"+1234567890", "+1234567890"},
		{"1234567890", "+1234567890"},
		{"123", "123"},
		{"", ""},
	}
	for _, tc := range tests {
		actual := cleanPhoneNumber(tc.input)
		if actual != tc.expected {
			t.Errorf("cleanPhoneNumber(%q) = %q, expected %q", tc.input, actual, tc.expected)
		}
	}

	// 2. formatDisplayName & formatChatSubtitle
	chatSavedDM := Chat{
		JID:  "1234567890@s.whatsapp.net",
		Name: "Alice Smith",
	}
	if name := formatDisplayName(chatSavedDM); name != "Alice Smith" {
		t.Errorf("formatDisplayName(chatSavedDM) = %q, expected %q", name, "Alice Smith")
	}
	if sub := formatChatSubtitle(chatSavedDM); sub != "+1234567890" {
		t.Errorf("formatChatSubtitle(chatSavedDM) = %q, expected %q", sub, "+1234567890")
	}

	chatUnsavedDM := Chat{
		JID:  "1234567890@s.whatsapp.net",
		Name: "",
	}
	if name := formatDisplayName(chatUnsavedDM); name != "+1234567890" {
		t.Errorf("formatDisplayName(chatUnsavedDM) = %q, expected %q", name, "+1234567890")
	}
	if sub := formatChatSubtitle(chatUnsavedDM); sub != "" {
		t.Errorf("formatChatSubtitle(chatUnsavedDM) = %q, expected empty string", sub)
	}

	chatUnsavedLIDMapped := Chat{
		JID:  "999888777@lid",
		Name: "+1987654321",
	}
	if name := formatDisplayName(chatUnsavedLIDMapped); name != "+1987654321" {
		t.Errorf("formatDisplayName(chatUnsavedLIDMapped) = %q, expected %q", name, "+1987654321")
	}

	chatUnsavedLIDUnmapped := Chat{
		JID:  "999888777@lid",
		Name: "",
	}
	if name := formatDisplayName(chatUnsavedLIDUnmapped); name != "WhatsApp User" {
		t.Errorf("formatDisplayName(chatUnsavedLIDUnmapped) = %q, expected %q", name, "WhatsApp User")
	}

	chatUnnamedGroup := Chat{
		JID:  "12345-67890@g.us",
		Name: "",
	}
	if name := formatDisplayName(chatUnnamedGroup); name != "Group" {
		t.Errorf("formatDisplayName(chatUnnamedGroup) = %q, expected %q", name, "Group")
	}

	chatNamedGroup := Chat{
		JID:  "12345-67890@g.us",
		Name: "Project Team",
	}
	if name := formatDisplayName(chatNamedGroup); name != "Project Team" {
		t.Errorf("formatDisplayName(chatNamedGroup) = %q, expected %q", name, "Project Team")
	}
}

func TestMediaHelpers(t *testing.T) {
	if s := formatFileSize(500); s != "500 B" {
		t.Errorf("formatFileSize(500) = %q, expected '500 B'", s)
	}
	if s := formatFileSize(1048576); s != "1.0 MB" {
		t.Errorf("formatFileSize(1048576) = %q, expected '1.0 MB'", s)
	}

	// Test parseE2EMessage
	imgMsg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption: proto.String("Test Photo"),
		},
	}
	raw, err := proto.Marshal(imgMsg)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	parsed := parseE2EMessage(raw)
	if parsed == nil || parsed.GetImageMessage() == nil || parsed.GetImageMessage().GetCaption() != "Test Photo" {
		t.Fatalf("unexpected parsed result: %+v", parsed)
	}
}

func TestSenderColorAndMentions(t *testing.T) {
	// 1. Test getSenderColor determinism
	c1 := getSenderColor("1234567890@s.whatsapp.net")
	c2 := getSenderColor("1234567890@s.whatsapp.net")
	if c1 != c2 {
		t.Errorf("getSenderColor not deterministic: %q != %q", c1, c2)
	}
	if len(c1) != 7 || c1[0] != '#' {
		t.Errorf("expected hex color, got %q", c1)
	}

	c3 := getSenderColor("9876543210@s.whatsapp.net")
	if c3 == "" {
		t.Errorf("expected valid color, got empty string")
	}

	// 2. Test formatMessageTextWithMentions without mentions
	markupNoMentions, hasMentions := formatMessageTextWithMentions("Hello world & friends <3", nil, nil)
	if hasMentions {
		t.Errorf("expected hasMentions=false")
	}
	if markupNoMentions != "Hello world &amp; friends &lt;3" {
		t.Errorf("expected escaped text, got %q", markupNoMentions)
	}

	// 3. Test formatMessageTextWithMentions with @ mentions
	resolver := func(jid string) string {
		if jid == "1234567890@s.whatsapp.net" {
			return "Alice"
		}
		return ""
	}
	text := "Hello @1234567890 and @9999999999 please review <this>"
	mentioned := []string{"1234567890@s.whatsapp.net"}
	markupWithMentions, has := formatMessageTextWithMentions(text, mentioned, resolver)
	if !has {
		t.Errorf("expected has=true for mentions")
	}
	expectedSubstring1 := `<a href="user:1234567890@s.whatsapp.net"><b>@Alice</b></a>`
	expectedSubstring2 := `<a href="user:9999999999@s.whatsapp.net"><b>@+9999999999</b></a>`
	expectedEscaped := `&lt;this&gt;`

	if !strings.Contains(markupWithMentions, expectedSubstring1) {
		t.Errorf("markup missing %q: got %q", expectedSubstring1, markupWithMentions)
	}
	if !strings.Contains(markupWithMentions, expectedSubstring2) {
		t.Errorf("markup missing %q: got %q", expectedSubstring2, markupWithMentions)
	}
	if !strings.Contains(markupWithMentions, expectedEscaped) {
		t.Errorf("markup missing %q: got %q", expectedEscaped, markupWithMentions)
	}
}

func TestGetChatOrContact(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	// Insert contact into whatsmeow_contacts
	_, err = db.Exec(`INSERT INTO whatsmeow_contacts (our_jid, their_jid, first_name, full_name)
		VALUES ('', '5551234@s.whatsapp.net', 'Bob', 'Bob Dylan')`)
	if err != nil {
		t.Fatalf("inserting contact: %v", err)
	}

	chat, err := store.getChatOrContact(ctx, "5551234@s.whatsapp.net")
	if err != nil {
		t.Fatalf("getChatOrContact: %v", err)
	}
	if chat.Name != "Bob Dylan" {
		t.Errorf("expected 'Bob Dylan', got %q", chat.Name)
	}
}

func TestSearchFilters(t *testing.T) {
	// Test chat user/phone filter
	chat := Chat{
		JID:  "1234567890@s.whatsapp.net",
		Name: "Alice Wonderland",
	}
	name := strings.ToLower(formatDisplayName(chat))
	phone := strings.ToLower(cleanPhoneNumber(chat.JID))

	matchesAlice := strings.Contains(name, "alice") || strings.Contains(phone, "alice")
	matchesNumber := strings.Contains(name, "12345") || strings.Contains(phone, "12345")
	matchesBob := strings.Contains(name, "bob") || strings.Contains(phone, "bob")

	if !matchesAlice {
		t.Errorf("expected chat to match 'alice'")
	}
	if !matchesNumber {
		t.Errorf("expected chat to match '12345'")
	}
	if matchesBob {
		t.Errorf("expected chat not to match 'bob'")
	}

	// Test message text filter
	msg := StoredMessage{
		SenderName: "Alice",
		Text:       "Let's meet tomorrow at noon",
	}
	qMatch := strings.Contains(strings.ToLower(msg.Text), "meet") || strings.Contains(strings.ToLower(msg.SenderName), "meet")
	qNoMatch := strings.Contains(strings.ToLower(msg.Text), "dinner") || strings.Contains(strings.ToLower(msg.SenderName), "dinner")

	if !qMatch {
		t.Errorf("expected message to match 'meet'")
	}
	if qNoMatch {
		t.Errorf("expected message not to match 'dinner'")
	}
}

func TestStarredAndTabbedMessages(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	chatJID := "123456@s.whatsapp.net"
	_, err = db.Exec(`INSERT INTO whatsdown_chats (jid, name, last_message_at) VALUES (?, 'Test Chat', 1000)`, chatJID)
	if err != nil {
		t.Fatalf("inserting chat: %v", err)
	}

	// Insert photo, audio, doc, text
	_, err = db.Exec(`
		INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, starred)
		VALUES
			(?, 'm-img', ?, 'Alice', 0, 1001, 'image', 'sunset.jpg', 0),
			(?, 'm-aud', ?, 'Alice', 0, 1002, 'audio', '', 0),
			(?, 'm-doc', ?, 'Alice', 0, 1003, 'document', 'report.pdf', 0),
			(?, 'm-txt', ?, 'Alice', 0, 1004, 'text', 'hello', 0)
	`, chatJID, chatJID, chatJID, chatJID, chatJID, chatJID, chatJID, chatJID)
	if err != nil {
		t.Fatalf("inserting messages: %v", err)
	}

	// Test listMediaMessages
	media, err := store.listMediaMessages(ctx, chatJID)
	if err != nil || len(media) != 1 || media[0].ID != "m-img" {
		t.Fatalf("expected 1 media message (m-img), got %d (%v)", len(media), err)
	}

	// Test listAudioMessages
	audio, err := store.listAudioMessages(ctx, chatJID)
	if err != nil || len(audio) != 1 || audio[0].ID != "m-aud" {
		t.Fatalf("expected 1 audio message (m-aud), got %d (%v)", len(audio), err)
	}

	// Test listDocMessages
	docs, err := store.listDocMessages(ctx, chatJID)
	if err != nil || len(docs) != 1 || docs[0].ID != "m-doc" {
		t.Fatalf("expected 1 doc message (m-doc), got %d (%v)", len(docs), err)
	}

	// Star m-txt and m-img
	if err := store.toggleMessageStarred(ctx, chatJID, "m-txt", true); err != nil {
		t.Fatalf("toggleMessageStarred m-txt: %v", err)
	}
	if err := store.toggleMessageStarred(ctx, chatJID, "m-img", true); err != nil {
		t.Fatalf("toggleMessageStarred m-img: %v", err)
	}

	starred, err := store.listStarredMessages(ctx, chatJID)
	if err != nil || len(starred) != 2 {
		t.Fatalf("expected 2 starred messages, got %d (%v)", len(starred), err)
	}

	// Unstar m-txt
	if err := store.toggleMessageStarred(ctx, chatJID, "m-txt", false); err != nil {
		t.Fatalf("unstar m-txt: %v", err)
	}
	starred, err = store.listStarredMessages(ctx, chatJID)
	if err != nil || len(starred) != 1 || starred[0].ID != "m-img" {
		t.Fatalf("expected 1 starred message (m-img), got %d (%v)", len(starred), err)
	}

	// Test LID <-> PN toggleMessageStarred and resolveLIDtoPN
	_, _ = db.ExecContext(ctx, `INSERT INTO whatsmeow_lid_map (lid, pn) VALUES ('220212351746073', '393478485526')`)
	if resolved := store.resolveLIDtoPN(ctx, "220212351746073@lid"); resolved != "393478485526@s.whatsapp.net" {
		t.Fatalf("expected resolved PN JID, got %s", resolved)
	}

	lidChat := "220212351746073@lid"
	_, _ = db.ExecContext(ctx, `
		INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text, raw, starred, status)
		VALUES (?, 'm-lid-1', ?, 'User', 0, 1005, 'text', 'hello', x'', 1, 0)
	`, lidChat, lidChat)

	// Unstar using PN JID when stored as LID
	if err := store.toggleMessageStarred(ctx, "393478485526@s.whatsapp.net", "m-lid-1", false); err != nil {
		t.Fatalf("toggleMessageStarred with PN for LID message: %v", err)
	}
	var starredVal int
	_ = db.QueryRowContext(ctx, `SELECT starred FROM whatsdown_messages WHERE id = 'm-lid-1'`).Scan(&starredVal)
	if starredVal != 0 {
		t.Fatalf("expected message to be unstarred (0), got %d", starredVal)
	}

	// Re-star using PN JID
	if err := store.toggleMessageStarred(ctx, "393478485526@s.whatsapp.net", "m-lid-1", true); err != nil {
		t.Fatalf("toggleMessageStarred re-star: %v", err)
	}
	_ = db.QueryRowContext(ctx, `SELECT starred FROM whatsdown_messages WHERE id = 'm-lid-1'`).Scan(&starredVal)
	if starredVal != 1 {
		t.Fatalf("expected message to be starred (1), got %d", starredVal)
	}
}

func TestGroupSenderResolution(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	groupJID := "987654321-123456@g.us"
	_, err = db.Exec(`INSERT INTO whatsdown_chats (jid, name, last_message_at) VALUES (?, 'Cool Group', 1000)`, groupJID)
	if err != nil {
		t.Fatalf("inserting chat: %v", err)
	}

	// 1. Saved contact
	_, _ = db.Exec(`INSERT INTO whatsmeow_contacts (our_jid, their_jid, first_name, full_name)
		VALUES ('', '1111111111@s.whatsapp.net', 'Alice', 'Alice Walker')`)

	// 2. User with push name
	_, _ = db.Exec(`INSERT INTO whatsmeow_contacts (our_jid, their_jid, push_name)
		VALUES ('', '2222222222@s.whatsapp.net', 'CharlieBrown')`)

	// 3. Mapped LID
	_, _ = db.Exec(`INSERT INTO whatsmeow_lid_map (lid, pn) VALUES ('100000000000001', '15553334444')`)

	// Insert messages in group
	_, err = db.Exec(`
		INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text)
		VALUES
			(?, 'msg-1', '1111111111@s.whatsapp.net', '', 0, 1001, 'text', 'from Alice'),
			(?, 'msg-2', '2222222222@s.whatsapp.net', '', 0, 1002, 'text', 'from Charlie'),
			(?, 'msg-3', '100000000000001@lid', '', 0, 1003, 'text', 'from mapped LID'),
			(?, 'msg-4', '100000000000002@lid', '', 0, 1004, 'text', 'from unmapped LID')
	`, groupJID, groupJID, groupJID, groupJID)
	if err != nil {
		t.Fatalf("inserting messages: %v", err)
	}

	msgs, err := store.listMessages(ctx, groupJID, time.Time{}, 10)
	if err != nil || len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d (%v)", len(msgs), err)
	}

	// Message 1: contact saved name "Alice Walker"
	if msgs[0].SenderName != "Alice Walker" {
		t.Errorf("msg-1 expected 'Alice Walker', got %q", msgs[0].SenderName)
	}
	if !msgs[0].SenderSaved {
		t.Errorf("msg-1 expected SenderSaved to be true for saved contact Alice Walker")
	}
	// Message 2: push name "CharlieBrown"
	if msgs[1].SenderName != "CharlieBrown" {
		t.Errorf("msg-2 expected 'CharlieBrown', got %q", msgs[1].SenderName)
	}
	if msgs[1].SenderSaved {
		t.Errorf("msg-2 expected SenderSaved to be false for push name only CharlieBrown")
	}
	// Message 3: mapped LID real phone number "+15553334444"
	if msgs[2].SenderName != "+15553334444" {
		t.Errorf("msg-3 expected '+15553334444', got %q", msgs[2].SenderName)
	}
	if msgs[2].SenderSaved {
		t.Errorf("msg-3 expected SenderSaved to be false for mapped LID without contact name")
	}
	// Message 4: unmapped LID should NEVER be a broken fake number "+100000000000002"
	if msgs[3].SenderName != "WhatsApp User" {
		t.Errorf("msg-4 expected 'WhatsApp User', got %q", msgs[3].SenderName)
	}
	if msgs[3].SenderSaved {
		t.Errorf("msg-4 expected SenderSaved to be false for unmapped LID")
	}
}

func TestCleanPhoneNumberLID(t *testing.T) {
	// Standard phone JID
	if clean := cleanPhoneNumber("15551234567@s.whatsapp.net"); clean != "+15551234567" {
		t.Errorf("expected '+15551234567', got %q", clean)
	}
	// LID JID must not have '+' prepended as fake number
	if clean := cleanPhoneNumber("100000000000001@lid"); clean != "WhatsApp User" {
		t.Errorf("expected 'WhatsApp User', got %q", clean)
	}
	// Raw 15-digit LID
	if clean := cleanPhoneNumber("100000000000001"); clean != "WhatsApp User" {
		t.Errorf("expected 'WhatsApp User', got %q", clean)
	}
}

func TestSenderSavedAndTildePrefix(t *testing.T) {
	// Simulate group message display formatting logic
	formatGroupSender := func(sender string, saved bool) string {
		if !saved && sender != "" && sender != "Member" && !strings.HasPrefix(sender, "~") {
			return "~" + sender
		}
		return sender
	}

	// Saved contact should not have tilde
	if got := formatGroupSender("Alice Walker", true); got != "Alice Walker" {
		t.Errorf("expected 'Alice Walker', got %q", got)
	}

	// Unsaved push name should have tilde
	if got := formatGroupSender("CharlieBrown", false); got != "~CharlieBrown" {
		t.Errorf("expected '~CharlieBrown', got %q", got)
	}

	// Unsaved phone number should have tilde
	if got := formatGroupSender("+15551234567", false); got != "~+15551234567" {
		t.Errorf("expected '~+15551234567', got %q", got)
	}

	// Already prefixed should not duplicate
	if got := formatGroupSender("~CharlieBrown", false); got != "~CharlieBrown" {
		t.Errorf("expected '~CharlieBrown', got %q", got)
	}
}

func TestSenderColorPalettes(t *testing.T) {
	if len(senderColorPalette) != len(senderColorPaletteDark) {
		t.Fatalf("expected palette lengths to match: light=%d, dark=%d", len(senderColorPalette), len(senderColorPaletteDark))
	}
	if len(senderColorPalette) < 10 {
		t.Errorf("expected at least 10 palette colors, got %d", len(senderColorPalette))
	}
	// Verify getSenderColor returns non-empty deterministic color
	c1 := getSenderColor("user-1234")
	c2 := getSenderColor("user-1234")
	if c1 == "" || c1 != c2 {
		t.Errorf("getSenderColor should be deterministic: %q vs %q", c1, c2)
	}
}

func TestMediaPreviewHelpers(t *testing.T) {
	// 1. stopCurrentAudio should be safe to call at any time
	stopCurrentAudio()
	stopCurrentAudio()

	// 2. Empty paths should return nil safely
	if tex := getPDFThumbnailTexture(""); tex != nil {
		t.Errorf("expected nil for empty PDF path, got %v", tex)
	}
	if tex := getVideoThumbnailTexture(""); tex != nil {
		t.Errorf("expected nil for empty video path, got %v", tex)
	}

	// 3. Non-existent file should return nil safely without panic
	if tex := getPDFThumbnailTexture("/tmp/non_existent_file.pdf"); tex != nil {
		t.Errorf("expected nil for non-existent PDF, got %v", tex)
	}
	if tex := getVideoThumbnailTexture("/tmp/non_existent_file.mp4"); tex != nil {
		t.Errorf("expected nil for non-existent video, got %v", tex)
	}
}

func TestSearchMatchNavigationLogic(t *testing.T) {
	// Test cyclic navigation index calculations
	navigate := func(cur, delta, total int) int {
		if total == 0 {
			return 0
		}
		return (cur + delta + total) % total
	}

	// 3 matches: indices 0, 1, 2
	total := 3
	cur := 2 // started at latest match

	// Next (+1) wraps from 2 to 0
	cur = navigate(cur, 1, total)
	if cur != 0 {
		t.Errorf("expected 0, got %d", cur)
	}

	// Next (+1) goes from 0 to 1
	cur = navigate(cur, 1, total)
	if cur != 1 {
		t.Errorf("expected 1, got %d", cur)
	}

	// Prev (-1) goes back to 0
	cur = navigate(cur, -1, total)
	if cur != 0 {
		t.Errorf("expected 0, got %d", cur)
	}

	// Prev (-1) wraps from 0 to 2
	cur = navigate(cur, -1, total)
	if cur != 2 {
		t.Errorf("expected 2, got %d", cur)
	}
}

func TestPollMessageCreationAndParsing(t *testing.T) {
	question := "What is your favorite color?"
	options := []string{"Red", "Green", "Blue"}

	var optProtos []*waE2E.PollCreationMessage_Option
	for _, opt := range options {
		optProtos = append(optProtos, &waE2E.PollCreationMessage_Option{OptionName: proto.String(opt)})
	}
	waMsg := &waE2E.Message{
		PollCreationMessage: &waE2E.PollCreationMessage{
			Name:                   proto.String(question),
			Options:                optProtos,
			SelectableOptionsCount: proto.Uint32(1),
		},
	}

	kind, title, ok := describeMessage(waMsg)
	if !ok || kind != messageKindPoll {
		t.Fatalf("expected messageKindPoll, got %q (ok=%v)", kind, ok)
	}
	if title != question {
		t.Errorf("expected title %q, got %q", question, title)
	}

	raw, err := proto.Marshal(waMsg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	parsed := parseE2EMessage(raw)
	if parsed == nil || parsed.GetPollCreationMessage() == nil {
		t.Fatalf("expected parsed poll message, got nil")
	}
	poll := parsed.GetPollCreationMessage()
	if poll.GetName() != question {
		t.Errorf("expected question %q, got %q", question, poll.GetName())
	}
	if len(poll.GetOptions()) != 3 {
		t.Errorf("expected 3 options, got %d", len(poll.GetOptions()))
	}
	if poll.GetOptions()[0].GetOptionName() != "Red" {
		t.Errorf("expected 'Red', got %q", poll.GetOptions()[0].GetOptionName())
	}
}

func TestPollVotingStore(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening in-memory db: %v", err)
	}
	defer db.Close()

	s, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	chatJID := "testchat@s.whatsapp.net"
	msgID := "poll_msg_1"

	// 1. Initial votes should be empty
	votes, err := s.getPollVotes(ctx, chatJID, msgID)
	if err != nil {
		t.Fatalf("getPollVotes failed: %v", err)
	}
	if len(votes) != 0 {
		t.Fatalf("expected empty votes, got %v", votes)
	}

	// 2. Single choice voting (allowMultiple = false)
	// Vote for "Option A"
	votes, err = s.togglePollVote(ctx, chatJID, msgID, "Option A", false)
	if err != nil {
		t.Fatalf("togglePollVote failed: %v", err)
	}
	if !votes["Option A"] || len(votes) != 1 {
		t.Fatalf("expected only Option A selected, got %v", votes)
	}

	// Vote for "Option B" (should switch to Option B)
	votes, err = s.togglePollVote(ctx, chatJID, msgID, "Option B", false)
	if err != nil {
		t.Fatalf("togglePollVote failed: %v", err)
	}
	if !votes["Option B"] || votes["Option A"] || len(votes) != 1 {
		t.Fatalf("expected only Option B selected, got %v", votes)
	}

	// 3. Multi choice voting (allowMultiple = true) on a different message
	msgIDMulti := "poll_msg_2"
	votes, err = s.togglePollVote(ctx, chatJID, msgIDMulti, "Choice 1", true)
	if err != nil {
		t.Fatalf("togglePollVote multi failed: %v", err)
	}
	if !votes["Choice 1"] {
		t.Fatalf("expected Choice 1 to be voted")
	}

	votes, err = s.togglePollVote(ctx, chatJID, msgIDMulti, "Choice 2", true)
	if err != nil {
		t.Fatalf("togglePollVote multi 2 failed: %v", err)
	}
	if !votes["Choice 1"] || !votes["Choice 2"] || len(votes) != 2 {
		t.Fatalf("expected Choice 1 and 2 to be selected, got %v", votes)
	}

	// Toggle Choice 1 off
	votes, err = s.togglePollVote(ctx, chatJID, msgIDMulti, "Choice 1", true)
	if err != nil {
		t.Fatalf("togglePollVote toggle off failed: %v", err)
	}
	if votes["Choice 1"] || !votes["Choice 2"] || len(votes) != 1 {
		t.Fatalf("expected only Choice 2 selected after toggle off, got %v", votes)
	}
}

func TestPDFPageHelpers(t *testing.T) {
	// Empty path should return 1 safely
	if count := getPDFPageCount(""); count != 1 {
		t.Errorf("expected 1 for empty path, got %d", count)
	}
	// Non-existent path should return 1 safely
	if count := getPDFPageCount("/nonexistent/path.pdf"); count != 1 {
		t.Errorf("expected 1 for nonexistent path, got %d", count)
	}
}

func TestPollResultsAndMultiVoters(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening in-memory db: %v", err)
	}
	defer db.Close()

	s, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	chatJID := "group1@g.us"
	msgID := "poll_msg_results"

	// Alice votes for "Red"
	aliceJID := "alice@s.whatsapp.net"
	_, err = s.togglePollVote(ctx, chatJID, msgID, "Red", true, aliceJID)
	if err != nil {
		t.Fatalf("alice vote: %v", err)
	}

	// Bob votes for "Red" and "Blue"
	bobJID := "bob@s.whatsapp.net"
	err = s.setPollVotesForVoter(ctx, chatJID, msgID, bobJID, []string{"Red", "Blue"})
	if err != nil {
		t.Fatalf("bob vote: %v", err)
	}

	// Check results
	results, err := s.getPollResults(ctx, chatJID, msgID)
	if err != nil {
		t.Fatalf("getPollResults: %v", err)
	}

	if results.TotalVoters != 2 {
		t.Errorf("expected 2 total voters, got %d", results.TotalVoters)
	}
	if results.VoteCounts["Red"] != 2 {
		t.Errorf("expected 2 votes for Red, got %d", results.VoteCounts["Red"])
	}
	if results.VoteCounts["Blue"] != 1 {
		t.Errorf("expected 1 vote for Blue, got %d", results.VoteCounts["Blue"])
	}

	// Alice votes map should only have Red
	aliceVotes, err := s.getPollVotes(ctx, chatJID, msgID, aliceJID)
	if err != nil || !aliceVotes["Red"] || aliceVotes["Blue"] {
		t.Errorf("unexpected alice votes: %v", aliceVotes)
	}

	// Bob votes map should have Red and Blue
	bobVotes, err := s.getPollVotes(ctx, chatJID, msgID, bobJID)
	if err != nil || !bobVotes["Red"] || !bobVotes["Blue"] {
		t.Errorf("unexpected bob votes: %v", bobVotes)
	}
}

func TestChatMute(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	s, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	chatJID := "1234567890@s.whatsapp.net"
	_, err = db.Exec(`INSERT INTO whatsdown_chats (jid, name, last_message_at, unread_count, archived, pinned, muted) VALUES (?, 'Test Chat', 1700000000, 0, 0, 0, 0)`, chatJID)
	if err != nil {
		t.Fatalf("insert chat: %v", err)
	}

	ch, err := s.getChatOrContact(ctx, chatJID)
	if err != nil || ch.Muted {
		t.Fatalf("expected muted=false, got muted=%v err=%v", ch.Muted, err)
	}

	// Mute chat
	if err := s.setChatMuted(ctx, chatJID, true); err != nil {
		t.Fatalf("setChatMuted true: %v", err)
	}

	ch, err = s.getChatOrContact(ctx, chatJID)
	if err != nil || !ch.Muted {
		t.Fatalf("expected muted=true, got muted=%v err=%v", ch.Muted, err)
	}

	chats, err := s.listChats(ctx)
	if err != nil || len(chats) != 1 || !chats[0].Muted {
		t.Fatalf("expected 1 muted chat in listChats, got %v", chats)
	}

	// Unmute chat
	if err := s.setChatMuted(ctx, chatJID, false); err != nil {
		t.Fatalf("setChatMuted false: %v", err)
	}

	ch, err = s.getChatOrContact(ctx, chatJID)
	if err != nil || ch.Muted {
		t.Fatalf("expected muted=false, got muted=%v err=%v", ch.Muted, err)
	}
}

func TestSelfChatCanonicalizationAndCleanup(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	// Create whatsmeow_device table
	_, err = db.Exec(`
		CREATE TABLE whatsmeow_device (
			jid TEXT PRIMARY KEY,
			lid TEXT,
			push_name TEXT
		);
		INSERT INTO whatsmeow_device (jid, lid, push_name)
		VALUES ('31629563452:52@s.whatsapp.net', '55684821442644:52@lid', 'Test User');
	`)
	if err != nil {
		t.Fatalf("create whatsmeow_device: %v", err)
	}

	s, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	// Verify isSelfJID and getCanonicalSelfJID
	if !s.isSelfJID(ctx, "31629563452@s.whatsapp.net") {
		t.Errorf("expected isSelfJID=true for PN")
	}
	if !s.isSelfJID(ctx, "55684821442644@lid") {
		t.Errorf("expected isSelfJID=true for LID")
	}
	if s.isSelfJID(ctx, "1234567890@s.whatsapp.net") {
		t.Errorf("expected isSelfJID=false for other user")
	}
	canonical := s.getCanonicalSelfJID(ctx)
	if canonical != "31629563452@s.whatsapp.net" {
		t.Errorf("expected canonical self JID 31629563452@s.whatsapp.net, got %s", canonical)
	}

	// Simulate duplicate chat created under LID
	_, err = db.Exec(`
		INSERT INTO whatsdown_chats (jid, name, last_message_at, unread_count, archived, pinned, muted)
		VALUES ('31629563452@s.whatsapp.net', 'Self PN', 1700000000, 0, 0, 0, 0),
		       ('55684821442644@lid', '', 1700000005, 0, 0, 0, 0);

		INSERT INTO whatsdown_messages (chat_jid, id, sender_jid, sender_name, from_me, timestamp, kind, text)
		VALUES ('31629563452@s.whatsapp.net', 'msg-pn-1', '31629563452@s.whatsapp.net', 'Me', 1, 1700000000, 'text', 'Note 1'),
		       ('55684821442644@lid', 'msg-lid-1', '55684821442644@lid', 'Me', 0, 1700000005, 'text', 'Note 2 from phone');
	`)
	if err != nil {
		t.Fatalf("insert test duplicate chats/messages: %v", err)
	}

	// Run cleanupSelfChats
	s.cleanupSelfChats(ctx)

	// Verify LID chat is gone
	var chatCount int
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM whatsdown_chats WHERE jid = '55684821442644@lid'`).Scan(&chatCount)
	if chatCount != 0 {
		t.Errorf("expected LID chat row deleted, found count=%d", chatCount)
	}

	// Verify all messages moved to canonical JID and from_me is 1
	var msgCount int
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM whatsdown_messages WHERE chat_jid = '31629563452@s.whatsapp.net' AND from_me = 1`).Scan(&msgCount)
	if msgCount != 2 {
		t.Errorf("expected 2 messages with from_me=1 in canonical self chat, found %d", msgCount)
	}

	// Test saving a live message sent from phone to self (which WhatsApp delivers with chat=LID)
	lidJID, _ := types.ParseJID("55684821442644:0@lid")
	liveMsg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     lidJID,
				Sender:   lidJID,
				IsFromMe: false,
			},
			ID:        "msg-live-self-1",
			PushName:  "Test User",
			Timestamp: time.Unix(1700000010, 0),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Live note to self"),
		},
	}

	if err := s.saveLiveMessage(ctx, liveMsg); err != nil {
		t.Fatalf("saveLiveMessage self: %v", err)
	}

	// Should be saved under canonical chat JID with from_me=1
	msg, err := s.getMessage(ctx, canonical, "msg-live-self-1")
	if err != nil || msg == nil {
		t.Fatalf("getMessage for live note: %v", err)
	}
	if !msg.FromMe {
		t.Errorf("expected live self message to have FromMe=true")
	}

	// Unread count should remain 0 for self chat
	selfChat, err := s.getChatOrContact(ctx, canonical)
	if err != nil || selfChat.UnreadCount != 0 {
		t.Errorf("expected unread_count=0 for self chat, got %d", selfChat.UnreadCount)
	}

	// Test getMessageByID
	msgByID, err := s.getMessageByID(ctx, "msg-live-self-1")
	if err != nil || msgByID == nil || msgByID.Text != "Live note to self" {
		t.Errorf("getMessageByID failed: %v, msg=%v", err, msgByID)
	}
}

func TestReplyContextAndSummarization(t *testing.T) {
	// 1. Text message with reply context
	replyMsg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String("I agree with you!"),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:    proto.String("orig-msg-123"),
				Participant: proto.String("15551234567@s.whatsapp.net"),
				QuotedMessage: &waE2E.Message{
					Conversation: proto.String("What do you think?"),
				},
			},
		},
	}

	raw, err := proto.Marshal(replyMsg)
	if err != nil {
		t.Fatalf("marshaling replyMsg: %v", err)
	}

	ctxInfo := extractContextInfo(raw)
	if ctxInfo == nil {
		t.Fatalf("expected non-nil ctxInfo")
	}
	if ctxInfo.GetStanzaID() != "orig-msg-123" {
		t.Errorf("expected stanzaID 'orig-msg-123', got %q", ctxInfo.GetStanzaID())
	}
	if ctxInfo.GetParticipant() != "15551234567@s.whatsapp.net" {
		t.Errorf("expected participant '15551234567@s.whatsapp.net', got %q", ctxInfo.GetParticipant())
	}
	quoted := ctxInfo.GetQuotedMessage()
	if quoted == nil || quoted.GetConversation() != "What do you think?" {
		t.Errorf("expected quoted conversation 'What do you think?', got %v", quoted)
	}

	// 2. Test summarizeProtoMessage for various types
	txt, kind := summarizeProtoMessage(quoted)
	if txt != "What do you think?" || kind != messageKindText {
		t.Errorf("summarize text expected ('What do you think?', text), got (%q, %q)", txt, kind)
	}

	imgProto := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption: proto.String("Sunset at beach"),
		},
	}
	txt, kind = summarizeProtoMessage(imgProto)
	if txt != "📷 Sunset at beach" || kind != messageKindImage {
		t.Errorf("summarize image expected ('📷 Sunset at beach', image), got (%q, %q)", txt, kind)
	}

	vidProto := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{},
	}
	txt, kind = summarizeProtoMessage(vidProto)
	if txt != "🎥 Video" || kind != messageKindVideo {
		t.Errorf("summarize video expected ('🎥 Video', video), got (%q, %q)", txt, kind)
	}

	audProto := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			PTT: proto.Bool(true),
		},
	}
	txt, kind = summarizeProtoMessage(audProto)
	if txt != "🎤 Voice message" || kind != messageKindVoice {
		t.Errorf("summarize audio voice expected ('🎤 Voice message', voice), got (%q, %q)", txt, kind)
	}

	docProto := &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			FileName: proto.String("invoice.pdf"),
		},
	}
	txt, kind = summarizeProtoMessage(docProto)
	if txt != "📄 invoice.pdf" || kind != messageKindDocument {
		t.Errorf("summarize document expected ('📄 invoice.pdf', doc), got (%q, %q)", txt, kind)
	}
}

func TestMessageStatusTracking(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	chatJID := types.NewJID("123456789", types.DefaultUserServer)
	msgID := "test-status-msg-1"

	msg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   chatJID,
				IsFromMe: true,
			},
			ID:        types.MessageID(msgID),
			Timestamp: time.Unix(1700000000, 0),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Outgoing status test"),
		},
	}

	inserted, err := store.saveMessageTxWithStar(ctx, store.db, msg, false)
	if err != nil || !inserted {
		t.Fatalf("saveMessageTxWithStar failed: inserted=%v, err=%v", inserted, err)
	}

	stored, err := store.getMessage(ctx, chatJID.String(), msgID)
	if err != nil {
		t.Fatalf("getMessage failed: %v", err)
	}
	if stored.Status != MessageStatusSent {
		t.Errorf("expected initial status %d (Sent), got %d", MessageStatusSent, stored.Status)
	}

	// Update to Delivered (2)
	if err := store.updateMessageStatuses(ctx, []string{msgID}, MessageStatusDelivered); err != nil {
		t.Fatalf("updateMessageStatuses to Delivered failed: %v", err)
	}
	stored, _ = store.getMessage(ctx, chatJID.String(), msgID)
	if stored.Status != MessageStatusDelivered {
		t.Errorf("expected status %d (Delivered), got %d", MessageStatusDelivered, stored.Status)
	}

	// Update to Read (3)
	if err := store.updateMessageStatuses(ctx, []string{msgID}, MessageStatusRead); err != nil {
		t.Fatalf("updateMessageStatuses to Read failed: %v", err)
	}
	stored, _ = store.getMessage(ctx, chatJID.String(), msgID)
	if stored.Status != MessageStatusRead {
		t.Errorf("expected status %d (Read), got %d", MessageStatusRead, stored.Status)
	}

	// Test non-regression: try setting Delivered on a Read message
	if err := store.updateMessageStatuses(ctx, []string{msgID}, MessageStatusDelivered); err != nil {
		t.Fatalf("updateMessageStatuses regression test failed: %v", err)
	}
	stored, _ = store.getMessage(ctx, chatJID.String(), msgID)
	if stored.Status != MessageStatusRead {
		t.Errorf("status regressed from Read to %d", stored.Status)
	}

	// Test listMessages preserves status
	msgs, err := store.listMessages(ctx, chatJID.String(), time.Time{}, 10)
	if err != nil || len(msgs) == 0 {
		t.Fatalf("listMessages failed: %v", err)
	}
	if msgs[0].Status != MessageStatusRead {
		t.Errorf("listMessages expected status %d, got %d", MessageStatusRead, msgs[0].Status)
	}
}

func TestFormatStatusMarkup(t *testing.T) {
	readMarkup := formatStatusMarkup(MessageStatusRead)
	if !strings.Contains(readMarkup, "#53bdeb") || !strings.Contains(readMarkup, "・・") {
		t.Errorf("unexpected read markup: %q", readMarkup)
	}

	delivMarkup := formatStatusMarkup(MessageStatusDelivered)
	if !strings.Contains(delivMarkup, "・・") || strings.Contains(delivMarkup, "#53bdeb") {
		t.Errorf("unexpected delivered markup: %q", delivMarkup)
	}

	sentMarkup := formatStatusMarkup(MessageStatusSent)
	if !strings.Contains(sentMarkup, "・") || strings.Contains(sentMarkup, "・・") {
		t.Errorf("unexpected sent markup: %q", sentMarkup)
	}
}

func TestStickerMediaHelpers(t *testing.T) {
	stickerProto := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:          proto.String("https://mmg.whatsapp.net/d/f/sticker.enc"),
			Mimetype:     proto.String("image/webp"),
			Width:        proto.Uint32(512),
			Height:       proto.Uint32(512),
			PngThumbnail: []byte("fake-png-thumbnail-bytes"),
		},
	}
	raw, err := proto.Marshal(stickerProto)
	if err != nil {
		t.Fatalf("marshal sticker: %v", err)
	}

	txt, kind := summarizeProtoMessage(stickerProto)
	if txt != "🏷️ Sticker" || kind != messageKindSticker {
		t.Errorf("summarize sticker expected ('🏷️ Sticker', sticker), got (%q, %q)", txt, kind)
	}

	msg := StoredMessage{
		ChatJID: "123456789@s.whatsapp.net",
		ID:      "TEST_STICKER_MSG_01",
		Kind:    messageKindSticker,
		Raw:     raw,
	}

	mediaPath := getLocalMediaPath(msg)
	if !strings.HasSuffix(mediaPath, ".webp") {
		t.Errorf("expected local media path to end in .webp, got %q", mediaPath)
	}

	thumbPath := getThumbCachePath(msg)
	if !strings.HasSuffix(thumbPath, ".png") {
		t.Errorf("expected thumb cache path to end in .png, got %q", thumbPath)
	}
}

func TestSavedStickersStore(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	// 1. Initial list should be empty
	stickers, err := store.listSavedStickers(ctx)
	if err != nil {
		t.Fatalf("listSavedStickers: %v", err)
	}
	if len(stickers) != 0 {
		t.Errorf("expected 0 saved stickers, got %d", len(stickers))
	}

	// 2. Save a sticker
	st1 := SavedSticker{
		ID:       "hash_abc123",
		FilePath: "/path/to/stk1.webp",
		FileHash: "hash_abc123",
		Width:    512,
		Height:   512,
		SavedAt:  time.Now(),
	}
	if err := store.saveSticker(ctx, st1); err != nil {
		t.Fatalf("saveSticker: %v", err)
	}

	// 3. Verify isStickerSaved by ID and Hash
	saved, err := store.isStickerSaved(ctx, "hash_abc123")
	if err != nil || !saved {
		t.Errorf("expected isStickerSaved=true, got %v (err: %v)", saved, err)
	}

	notSaved, err := store.isStickerSaved(ctx, "non_existent_hash")
	if err != nil || notSaved {
		t.Errorf("expected isStickerSaved=false for nonexistent, got %v", notSaved)
	}

	// 4. Save a second sticker and check ordering (newest first)
	st2 := SavedSticker{
		ID:       "hash_def456",
		FilePath: "/path/to/stk2.webp",
		FileHash: "hash_def456",
		Width:    512,
		Height:   512,
		SavedAt:  time.Now().Add(5 * time.Second),
	}
	if err := store.saveSticker(ctx, st2); err != nil {
		t.Fatalf("saveSticker st2: %v", err)
	}

	stickers, err = store.listSavedStickers(ctx)
	if err != nil || len(stickers) != 2 {
		t.Fatalf("expected 2 saved stickers, got %d (err: %v)", len(stickers), err)
	}
	if stickers[0].ID != "hash_def456" {
		t.Errorf("expected newest sticker 'hash_def456' first, got %q", stickers[0].ID)
	}

	// 5. Delete sticker
	if err := store.deleteSavedSticker(ctx, "hash_abc123"); err != nil {
		t.Fatalf("deleteSavedSticker: %v", err)
	}
	saved, err = store.isStickerSaved(ctx, "hash_abc123")
	if err != nil || saved {
		t.Errorf("expected sticker to be deleted, but isStickerSaved=%v", saved)
	}

	stickers, err = store.listSavedStickers(ctx)
	if err != nil || len(stickers) != 1 {
		t.Errorf("expected 1 sticker remaining, got %d", len(stickers))
	}
}

func TestMessageEditStore(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening in-memory db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	chatJID := "1234567890@s.whatsapp.net"
	msgID := "MSG_EDIT_TEST_1"

	// Insert initial message
	_, err = db.ExecContext(ctx, `
		INSERT INTO whatsdown_messages (id, chat_jid, sender_jid, timestamp, from_me, text, kind, status)
		VALUES (?, ?, ?, ?, 1, ?, 'text', 1)
	`, msgID, chatJID, chatJID, time.Now().Unix(), "Initial original text")
	if err != nil {
		t.Fatalf("insert message: %v", err)
	}

	// 1. Perform first edit
	if err := store.saveMessageEdit(ctx, chatJID, msgID, "First edit text"); err != nil {
		t.Fatalf("saveMessageEdit 1: %v", err)
	}

	// Verify message in whatsdown_messages is updated
	var currentText string
	var edited int
	err = db.QueryRowContext(ctx, `SELECT text, edited FROM whatsdown_messages WHERE chat_jid = ? AND id = ?`, chatJID, msgID).Scan(&currentText, &edited)
	if err != nil {
		t.Fatalf("query updated message: %v", err)
	}
	if currentText != "First edit text" || edited != 1 {
		t.Errorf("expected updated message text 'First edit text' and edited=1, got %q, %d", currentText, edited)
	}

	// Verify edit history has the original version
	edits, err := store.getMessageEdits(ctx, chatJID, msgID)
	if err != nil {
		t.Fatalf("getMessageEdits: %v", err)
	}
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit in history, got %d", len(edits))
	}
	if edits[0].Text != "Initial original text" {
		t.Errorf("expected history text 'Initial original text', got %q", edits[0].Text)
	}

	// 2. Perform second edit
	if err := store.saveMessageEdit(ctx, chatJID, msgID, "Second edit text (latest)"); err != nil {
		t.Fatalf("saveMessageEdit 2: %v", err)
	}

	edits, err = store.getMessageEdits(ctx, chatJID, msgID)
	if err != nil {
		t.Fatalf("getMessageEdits 2: %v", err)
	}
	if len(edits) != 2 {
		t.Fatalf("expected 2 edits in history, got %d", len(edits))
	}
	if edits[0].Text != "Initial original text" || edits[1].Text != "First edit text" {
		t.Errorf("unexpected edit history: %+v", edits)
	}
}

func TestChatLastMessage(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	dmJID := types.NewJID("111222333", types.DefaultUserServer)
	groupJID := types.NewJID("999888777", types.GroupServer)
	aliceJID := types.NewJID("444555666", types.DefaultUserServer)

	// 1. Message in DM from other user
	msg1 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     dmJID,
				Sender:   dmJID,
				IsFromMe: false,
			},
			ID:        "dm-1",
			PushName:  "Bob",
			Timestamp: time.Unix(100, 0),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Hey there!"),
		},
	}
	if err := store.saveLiveMessage(ctx, msg1); err != nil {
		t.Fatalf("saveLiveMessage dm1: %v", err)
	}

	chats, err := store.listChats(ctx)
	if err != nil || len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d, err: %v", len(chats), err)
	}
	if chats[0].LastMessageText != "Hey there!" {
		t.Errorf("expected LastMessageText 'Hey there!', got %q", chats[0].LastMessageText)
	}
	if chats[0].LastMessageFromMe {
		t.Errorf("expected LastMessageFromMe false, got true")
	}
	if sub := formatChatSubtitle(chats[0]); sub != "Hey there!" {
		t.Errorf("expected subtitle 'Hey there!', got %q", sub)
	}

	// 2. Reply in DM from me
	msg2 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     dmJID,
				Sender:   dmJID,
				IsFromMe: true,
			},
			ID:        "dm-2",
			PushName:  "Me",
			Timestamp: time.Unix(200, 0),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("I am doing well, thanks!"),
		},
	}
	if err := store.saveLiveMessage(ctx, msg2); err != nil {
		t.Fatalf("saveLiveMessage dm2: %v", err)
	}

	chats, _ = store.listChats(ctx)
	if chats[0].LastMessageText != "I am doing well, thanks!" {
		t.Errorf("expected latest LastMessageText, got %q", chats[0].LastMessageText)
	}
	if !chats[0].LastMessageFromMe {
		t.Errorf("expected LastMessageFromMe true, got false")
	}
	if sub := formatChatSubtitle(chats[0]); sub != "You: I am doing well, thanks!" {
		t.Errorf("expected subtitle 'You: I am doing well, thanks!', got %q", sub)
	}

	// 3. Group message from Alice
	msg3 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     groupJID,
				Sender:   aliceJID,
				IsFromMe: false,
			},
			ID:        "grp-1",
			PushName:  "Alice",
			Timestamp: time.Unix(300, 0),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Meeting at 5pm"),
		},
	}
	if err := store.saveLiveMessage(ctx, msg3); err != nil {
		t.Fatalf("saveLiveMessage grp1: %v", err)
	}

	grpChat, err := store.getChatOrContact(ctx, groupJID.String())
	if err != nil {
		t.Fatalf("getChatOrContact: %v", err)
	}
	if grpChat.LastMessageText != "Meeting at 5pm" {
		t.Errorf("expected group LastMessageText 'Meeting at 5pm', got %q", grpChat.LastMessageText)
	}
	if grpChat.LastMessageSender != "Alice" {
		t.Errorf("expected group LastMessageSender 'Alice', got %q", grpChat.LastMessageSender)
	}
	if sub := formatChatSubtitle(grpChat); sub != "Alice: Meeting at 5pm" {
		t.Errorf("expected group subtitle 'Alice: Meeting at 5pm', got %q", sub)
	}

	// 4. Group message from me
	msg4 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     groupJID,
				Sender:   aliceJID,
				IsFromMe: true,
			},
			ID:        "grp-2",
			PushName:  "Me",
			Timestamp: time.Unix(400, 0),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("I will be there!"),
		},
	}
	if err := store.saveLiveMessage(ctx, msg4); err != nil {
		t.Fatalf("saveLiveMessage grp2: %v", err)
	}

	grpChat, _ = store.getChatOrContact(ctx, groupJID.String())
	if sub := formatChatSubtitle(grpChat); sub != "You: I will be there!" {
		t.Errorf("expected group subtitle 'You: I will be there!', got %q", sub)
	}

	// 5. Media message preview (Photo)
	msg5 := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     dmJID,
				Sender:   dmJID,
				IsFromMe: false,
			},
			ID:        "dm-3",
			PushName:  "Bob",
			Timestamp: time.Unix(500, 0),
		},
		Message: &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption: proto.String("Check this sunset!"),
			},
		},
	}
	if err := store.saveLiveMessage(ctx, msg5); err != nil {
		t.Fatalf("saveLiveMessage dm3: %v", err)
	}
	chats, _ = store.listChats(ctx)
	// Find dmJID chat
	for _, c := range chats {
		if c.JID == dmJID.String() {
			if sub := formatChatSubtitle(c); sub != "📷 Check this sunset!" {
				t.Errorf("expected image subtitle '📷 Check this sunset!', got %q", sub)
			}
			break
		}
	}
}

func TestImportStickersFromChatHistory(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	chatJID := types.NewJID("123456789", types.DefaultUserServer)

	// Create dummy sticker data and file
	sha := []byte("12345678901234567890123456789012")
	hash := "3132333435363738393031323334353637383930313233343536373839303132"
	destDir := savedStickersDir()
	destFile := destDir + "/" + hash + ".webp"
	_ = os.WriteFile(destFile, []byte("fake-webp-data"), 0o600)
	defer os.Remove(destFile)

	stickerMsg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   chatJID,
				IsFromMe: true,
			},
			ID:        "msg-sticker-1",
			Timestamp: time.Unix(1700000000, 0),
		},
		Message: &waE2E.Message{
			StickerMessage: &waE2E.StickerMessage{
				URL:        proto.String("https://example.com/sticker"),
				FileSHA256: sha,
				Width:      proto.Uint32(512),
				Height:     proto.Uint32(512),
			},
		},
	}

	if err := store.saveLiveMessage(ctx, stickerMsg); err != nil {
		t.Fatalf("saveLiveMessage: %v", err)
	}

	imported := store.importStickersFromChatHistory(ctx, nil)
	if imported != 1 {
		t.Fatalf("expected 1 imported sticker, got %d", imported)
	}

	saved, err := store.isStickerSaved(ctx, hash)
	if err != nil || !saved {
		t.Fatalf("expected sticker %s to be saved, saved=%v, err=%v", hash, saved, err)
	}

	stickers, err := store.listSavedStickers(ctx)
	if err != nil || len(stickers) != 1 {
		t.Fatalf("expected 1 saved sticker in DB, got %d (err: %v)", len(stickers), err)
	}
	if stickers[0].Width != 512 || stickers[0].Height != 512 {
		t.Errorf("expected 512x512, got %dx%d", stickers[0].Width, stickers[0].Height)
	}

	// Running import again should skip already saved sticker
	importedAgain := store.importStickersFromChatHistory(ctx, nil)
	if importedAgain != 0 {
		t.Errorf("expected 0 imported on re-run, got %d", importedAgain)
	}
}

func TestCommunityGrouping(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	commJID := types.NewJID("1111111111", "g.us")
	sub1JID := types.NewJID("2222222222", "g.us")
	sub2JID := types.NewJID("3333333333", "g.us")
	regJID := types.NewJID("4444444444", "g.us")

	groups := []*types.GroupInfo{
		{
			JID: commJID,
			GroupName: types.GroupName{
				Name: "Tech Community",
			},
			GroupParent: types.GroupParent{
				IsParent: true,
			},
		},
		{
			JID: sub1JID,
			GroupName: types.GroupName{
				Name: "Announcements",
			},
			GroupLinkedParent: types.GroupLinkedParent{
				LinkedParentJID: commJID,
			},
			GroupIsDefaultSub: types.GroupIsDefaultSub{
				IsDefaultSubGroup: true,
			},
		},
		{
			JID: sub2JID,
			GroupName: types.GroupName{
				Name: "General Chat",
			},
			GroupLinkedParent: types.GroupLinkedParent{
				LinkedParentJID: commJID,
			},
		},
		{
			JID: regJID,
			GroupName: types.GroupName{
				Name: "Standalone Project",
			},
		},
	}

	if err := store.syncGroups(ctx, groups); err != nil {
		t.Fatalf("syncGroups failed: %v", err)
	}

	chats, err := store.listChats(ctx)
	if err != nil {
		t.Fatalf("listChats failed: %v", err)
	}

	chatMap := make(map[string]Chat)
	for _, c := range chats {
		chatMap[c.JID] = c
	}

	comm, ok := chatMap[commJID.String()]
	if !ok {
		t.Fatalf("expected community chat %s to exist", commJID)
	}
	if !comm.IsParent {
		t.Errorf("expected comm.IsParent to be true")
	}
	if comm.ParentJID != "" {
		t.Errorf("expected comm.ParentJID to be empty, got %q", comm.ParentJID)
	}

	sub1, ok := chatMap[sub1JID.String()]
	if !ok {
		t.Fatalf("expected sub1 chat %s to exist", sub1JID)
	}
	if sub1.ParentJID != commJID.String() {
		t.Errorf("expected sub1.ParentJID to be %q, got %q", commJID.String(), sub1.ParentJID)
	}
	if !sub1.IsDefaultSubGroup {
		t.Errorf("expected sub1.IsDefaultSubGroup to be true")
	}

	sub2, ok := chatMap[sub2JID.String()]
	if !ok {
		t.Fatalf("expected sub2 chat %s to exist", sub2JID)
	}
	if sub2.ParentJID != commJID.String() {
		t.Errorf("expected sub2.ParentJID to be %q, got %q", commJID.String(), sub2.ParentJID)
	}

	reg, ok := chatMap[regJID.String()]
	if !ok {
		t.Fatalf("expected reg chat %s to exist", regJID)
	}
	if reg.ParentJID != "" || reg.IsParent {
		t.Errorf("expected regular chat to not be parent and have empty ParentJID")
	}

	// Test setChatCommunity
	if err := store.setChatCommunity(ctx, regJID.String(), commJID.String(), false, false); err != nil {
		t.Fatalf("setChatCommunity failed: %v", err)
	}
	updatedReg, err := store.getChatOrContact(ctx, regJID.String())
	if err != nil {
		t.Fatalf("getChatOrContact failed: %v", err)
	}
	if updatedReg.ParentJID != commJID.String() {
		t.Errorf("expected updatedReg.ParentJID to be %q, got %q", commJID.String(), updatedReg.ParentJID)
	}
}

func TestCallHistory(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	// 1. Direct call saving and querying
	c1 := CallRecord{
		ID:           "call-1",
		ChatJID:      "15550001@s.whatsapp.net",
		CallerJID:    "15550001@s.whatsapp.net",
		CallerName:   "Alice",
		Timestamp:    time.Now().Add(-10 * time.Minute),
		DurationSecs: 0,
		IsVideo:      false,
		Outcome:      CallOutcomeMissed,
		FromMe:       false,
	}
	c2 := CallRecord{
		ID:           "call-2",
		ChatJID:      "15550002@s.whatsapp.net",
		CallerJID:    "15550002@s.whatsapp.net",
		CallerName:   "Bob",
		Timestamp:    time.Now().Add(-5 * time.Minute),
		DurationSecs: 154,
		IsVideo:      true,
		Outcome:      CallOutcomeConnected,
		FromMe:       true,
	}

	if err := store.saveCall(ctx, c1); err != nil {
		t.Fatalf("saveCall c1 failed: %v", err)
	}
	if err := store.saveCall(ctx, c2); err != nil {
		t.Fatalf("saveCall c2 failed: %v", err)
	}

	calls, err := store.listCalls(ctx, "")
	if err != nil || len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d (%v)", len(calls), err)
	}
	// Ordered by timestamp DESC: c2 is newer than c1
	if calls[0].ID != "call-2" || calls[1].ID != "call-1" {
		t.Errorf("unexpected call order: %s, %s", calls[0].ID, calls[1].ID)
	}
	if !calls[0].IsVideo || calls[0].DurationSecs != 154 {
		t.Errorf("call-2 properties mismatch: video=%v, duration=%d", calls[0].IsVideo, calls[0].DurationSecs)
	}

	// Filter by search query
	bobCalls, err := store.listCalls(ctx, "Bob")
	if err != nil || len(bobCalls) != 1 {
		t.Fatalf("expected 1 call for 'Bob', got %d (%v)", len(bobCalls), err)
	}

	// Delete single call
	if err := store.deleteCall(ctx, "call-1"); err != nil {
		t.Fatalf("deleteCall failed: %v", err)
	}
	callsAfterDel, _ := store.listCalls(ctx, "")
	if len(callsAfterDel) != 1 || callsAfterDel[0].ID != "call-2" {
		t.Fatalf("expected 1 call after delete, got %d", len(callsAfterDel))
	}

	// 2. Call extraction from CallLogMesssage inside message
	isVideo := true
	outcome := waE2E.CallLogMessage_MISSED
	dur := int64(0)
	chatJID := types.NewJID("15559999", "s.whatsapp.net")
	msg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     chatJID,
				Sender:   chatJID,
				IsFromMe: false,
			},
			ID:        "msg-call-1",
			Timestamp: time.Now(),
			PushName:  "Charlie",
		},
		Message: &waE2E.Message{
			CallLogMesssage: &waE2E.CallLogMessage{
				IsVideo:      &isVideo,
				CallOutcome:  &outcome,
				DurationSecs: &dur,
			},
		},
	}

	kind, text, ok := describeMessage(msg.Message)
	if !ok || kind != "call" {
		t.Fatalf("describeMessage expected call, got kind=%q, ok=%v", kind, ok)
	}
	if text != "Missed video call" {
		t.Errorf("expected 'Missed video call', got %q", text)
	}

	if err := store.saveLiveMessage(ctx, msg); err != nil {
		t.Fatalf("saveLiveMessage with call failed: %v", err)
	}

	callRecs, err := store.listCalls(ctx, "msg-call-1")
	if err != nil || len(callRecs) != 1 {
		// Or search by caller name "Charlie"
		callRecs, err = store.listCalls(ctx, "Charlie")
	}
	if err != nil || len(callRecs) == 0 {
		t.Fatalf("expected call record from CallLogMessage to be saved, err: %v", err)
	}
	if !callRecs[0].IsVideo || callRecs[0].Outcome != CallOutcomeMissed {
		t.Errorf("extracted call properties mismatch: video=%v, outcome=%d", callRecs[0].IsVideo, callRecs[0].Outcome)
	}

	// Clear calls
	if err := store.clearCalls(ctx); err != nil {
		t.Fatalf("clearCalls failed: %v", err)
	}
	callsCleared, _ := store.listCalls(ctx, "")
	if len(callsCleared) != 0 {
		t.Errorf("expected 0 calls after clearCalls, got %d", len(callsCleared))
	}
}

func TestReactionsStore(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}
	defer db.Close()

	store, err := openMessageStore(ctx, db)
	if err != nil {
		t.Fatalf("openMessageStore: %v", err)
	}

	chatJID := "testchat@s.whatsapp.net"
	msgID := "msg-react-1"

	// 1. Initial reactions should be empty
	reacts, err := store.getReactionsForMessage(ctx, chatJID, msgID)
	if err != nil {
		t.Fatalf("getReactionsForMessage failed: %v", err)
	}
	if len(reacts) != 0 {
		t.Errorf("expected 0 reactions, got %d", len(reacts))
	}

	// 2. Add reaction from contact
	r1 := Reaction{
		ChatJID:    chatJID,
		MessageID:  msgID,
		SenderJID:  "alice@s.whatsapp.net",
		SenderName: "Alice",
		Emoji:      "❤️",
		FromMe:     false,
		Timestamp:  time.Now(),
	}
	if err := store.saveReaction(ctx, r1); err != nil {
		t.Fatalf("saveReaction r1 failed: %v", err)
	}

	// 3. Add reaction from self
	r2 := Reaction{
		ChatJID:    chatJID,
		MessageID:  msgID,
		SenderJID:  "me@s.whatsapp.net",
		SenderName: "You",
		Emoji:      "👍",
		FromMe:     true,
		Timestamp:  time.Now(),
	}
	if err := store.saveReaction(ctx, r2); err != nil {
		t.Fatalf("saveReaction r2 failed: %v", err)
	}

	// 4. Verify getReactionsForMessage
	reacts, err = store.getReactionsForMessage(ctx, chatJID, msgID)
	if err != nil || len(reacts) != 2 {
		t.Fatalf("expected 2 reactions, got %d (err: %v)", len(reacts), err)
	}

	// 5. Verify getReactionsForChat
	rMap, err := store.getReactionsForChat(ctx, chatJID)
	if err != nil || len(rMap[msgID]) != 2 {
		t.Fatalf("expected 2 reactions in getReactionsForChat, got %d", len(rMap[msgID]))
	}

	// 6. Update reaction (Alice changes ❤️ to 😂)
	r1Updated := r1
	r1Updated.Emoji = "😂"
	if err := store.saveReaction(ctx, r1Updated); err != nil {
		t.Fatalf("saveReaction update failed: %v", err)
	}
	reacts, err = store.getReactionsForMessage(ctx, chatJID, msgID)
	if err != nil || len(reacts) != 2 {
		t.Fatalf("expected still 2 reactions after update, got %d", len(reacts))
	}
	var foundAliceEmoji string
	for _, r := range reacts {
		if r.SenderJID == "alice@s.whatsapp.net" {
			foundAliceEmoji = r.Emoji
		}
	}
	if foundAliceEmoji != "😂" {
		t.Errorf("expected Alice emoji to be 😂, got %s", foundAliceEmoji)
	}

	// 7. Remove my reaction
	if err := store.removeMyReaction(ctx, chatJID, msgID); err != nil {
		t.Fatalf("removeMyReaction failed: %v", err)
	}
	reacts, _ = store.getReactionsForMessage(ctx, chatJID, msgID)
	if len(reacts) != 1 || reacts[0].SenderJID != "alice@s.whatsapp.net" {
		t.Errorf("expected 1 reaction left from Alice, got %+v", reacts)
	}

	// 8. Remove Alice reaction
	if err := store.removeReaction(ctx, chatJID, msgID, "alice@s.whatsapp.net"); err != nil {
		t.Fatalf("removeReaction failed: %v", err)
	}
	reacts, _ = store.getReactionsForMessage(ctx, chatJID, msgID)
	if len(reacts) != 0 {
		t.Errorf("expected 0 reactions after removal, got %d", len(reacts))
	}

	// 9. Test saveLiveMessage with waE2E.ReactionMessage
	cTargetJID := types.NewJID("testchat", types.DefaultUserServer)
	cBobJID := types.NewJID("bob", types.DefaultUserServer)

	// First insert target text message
	targetMsgEvt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     cTargetJID,
				Sender:   cTargetJID,
				IsFromMe: false,
			},
			ID:        "target-msg-99",
			PushName:  "Test",
			Timestamp: time.Now(),
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Target Message"),
		},
	}
	if err := store.saveLiveMessage(ctx, targetMsgEvt); err != nil {
		t.Fatalf("saving target message failed: %v", err)
	}

	// Now Bob sends a reaction event
	reactionEvt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:     cTargetJID,
				Sender:   cBobJID,
				IsFromMe: false,
			},
			ID:        "reaction-msg-1",
			PushName:  "Bob",
			Timestamp: time.Now(),
		},
		Message: &waE2E.Message{
			ReactionMessage: &waE2E.ReactionMessage{
				Key: &waCommon.MessageKey{
					ID:        proto.String("target-msg-99"),
					RemoteJID: proto.String(cTargetJID.String()),
				},
				Text: proto.String("🎉"),
			},
		},
	}
	if err := store.saveLiveMessage(ctx, reactionEvt); err != nil {
		t.Fatalf("saving reaction event failed: %v", err)
	}

	// Check target message via listMessages
	msgs, err := store.listMessages(ctx, cTargetJID.String(), time.Time{}, 10)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("expected 1 message in listMessages, got %d", len(msgs))
	}
	if len(msgs[0].Reactions) != 1 || msgs[0].Reactions[0].Emoji != "🎉" {
		t.Errorf("expected 1 reaction with 🎉 on target message, got %+v", msgs[0].Reactions)
	}
}

func TestSameMinuteAndGrouping(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		loc = time.FixedZone("CEST", 2*3600)
	}
	tLocal := time.Date(2026, 10, 20, 14, 30, 15, 0, loc)

	ctrl := &whatsDownMainController{}
	m1 := StoredMessage{
		ChatJID:   "123@s.whatsapp.net",
		FromMe:    true,
		Timestamp: tLocal,
	}
	m2 := StoredMessage{
		ChatJID:   "123@s.whatsapp.net",
		FromMe:    true,
		Timestamp: tLocal.Add(20 * time.Second), // same minute 14:30
	}
	mDiffTime := StoredMessage{
		ChatJID:   "123@s.whatsapp.net",
		FromMe:    true,
		Timestamp: tLocal.Add(70 * time.Second), // 14:31
	}
	mDiffSender := StoredMessage{
		ChatJID:   "123@s.whatsapp.net",
		FromMe:    false,
		Timestamp: tLocal.Add(10 * time.Second),
	}

	if !ctrl.isSameAuthorAndDisplayTime(m1, m2) {
		t.Errorf("expected isSameAuthorAndDisplayTime to be true for consecutive messages from same sender with same displayed time")
	}
	if ctrl.isSameAuthorAndDisplayTime(m1, mDiffTime) {
		t.Errorf("expected isSameAuthorAndDisplayTime to be false for different displayed times")
	}
	if ctrl.isSameAuthorAndDisplayTime(m1, mDiffSender) {
		t.Errorf("expected isSameAuthorAndDisplayTime to be false for different senders")
	}
}
