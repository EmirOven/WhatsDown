package ui

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestForwardMessagePreservesMediaAndClearsReply(t *testing.T) {
	original := &waE2E.Message{ImageMessage: &waE2E.ImageMessage{URL: proto.String("https://example.test/media"), MediaKey: []byte{1, 2, 3}, Caption: proto.String("photo"), ContextInfo: &waE2E.ContextInfo{StanzaID: proto.String("quoted"), Participant: proto.String("sender"), ForwardingScore: proto.Uint32(2), MentionedJID: []string{"123@s.whatsapp.net"}}}}
	raw, _ := proto.Marshal(original)
	forwarded, err := buildForwardMessage(store.StoredMessage{Kind: store.MessageKindImage, Raw: raw})
	if err != nil {
		t.Fatal(err)
	}
	image := forwarded.GetImageMessage()
	info := image.GetContextInfo()
	if image.GetURL() != original.ImageMessage.GetURL() || len(image.MediaKey) != 3 || !info.GetIsForwarded() || info.GetForwardingScore() != 3 || info.GetStanzaID() != "" || len(info.MentionedJID) != 1 {
		t.Fatalf("incorrect forward: %v", forwarded)
	}
	if original.ImageMessage.ContextInfo.GetStanzaID() != "quoted" {
		t.Fatal("source mutated")
	}
}

func TestForwardTextAndRejectUnavailable(t *testing.T) {
	forwarded, err := buildForwardMessage(store.StoredMessage{Kind: store.MessageKindText, Text: "hello"})
	if err != nil || forwarded.GetExtendedTextMessage().GetText() != "hello" {
		t.Fatalf("forwarded text: %v, %v", forwarded, err)
	}
	for _, msg := range []store.StoredMessage{{Kind: store.MessageKindDeleted}, {Kind: store.MessageKindImage}} {
		if _, err := buildForwardMessage(msg); err == nil {
			t.Fatal("unavailable message accepted")
		}
	}
}

func TestCaptionMentionsPreserveReply(t *testing.T) {
	saved := activeMentionState.insertedMentions
	t.Cleanup(func() { activeMentionState.insertedMentions = saved })
	activeMentionState.insertedMentions = map[string]string{"Alice": "123@s.whatsapp.net"}
	c := &MainController{currentChat: &store.Chat{JID: "123@g.us"}}
	reply := &waE2E.ContextInfo{StanzaID: proto.String("reply-id")}
	text, info := c.prepareMentionsContext("Hi @Alice", reply)
	if text != "Hi @123" || info.GetStanzaID() != "reply-id" || len(info.MentionedJID) != 1 {
		t.Fatalf("caption context: %q, %v", text, info)
	}
}
