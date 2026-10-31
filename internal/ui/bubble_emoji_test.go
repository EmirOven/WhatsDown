package ui

import (
	"os"
	"testing"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestIsSingleEmojiMessage(t *testing.T) {
	c := &MainController{}

	singleMsg := store.StoredMessage{Kind: store.MessageKindText, Text: "😀"}
	if !c.isSingleEmojiMessage(singleMsg) {
		t.Errorf("expected isSingleEmojiMessage to be true for single emoji message")
	}

	multiMsg := store.StoredMessage{Kind: store.MessageKindText, Text: "😀😀"}
	if c.isSingleEmojiMessage(multiMsg) {
		t.Errorf("expected isSingleEmojiMessage to be false for multiple emojis")
	}

	textMsg := store.StoredMessage{Kind: store.MessageKindText, Text: "Hello 😀"}
	if c.isSingleEmojiMessage(textMsg) {
		t.Errorf("expected isSingleEmojiMessage to be false for text with emoji")
	}

	imageMsg := store.StoredMessage{Kind: store.MessageKindImage, Text: "😀"}
	if c.isSingleEmojiMessage(imageMsg) {
		t.Errorf("expected isSingleEmojiMessage to be false for media kind")
	}

	ctxInfo := &waE2E.ContextInfo{StanzaID: proto.String("quoted123")}
	protoMsg := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String("😀"), ContextInfo: ctxInfo}}
	raw, _ := proto.Marshal(protoMsg)
	quotedMsg := store.StoredMessage{Kind: store.MessageKindText, Text: "😀", Raw: raw}
	if c.isSingleEmojiMessage(quotedMsg) {
		t.Errorf("expected isSingleEmojiMessage to be false for quoted reply")
	}
}

func TestBuildBubbleTextSingleEmoji(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("skipping UI test in headless environment without display")
	}
	gtk.Init()

	c := &MainController{}

	singleMsg := store.StoredMessage{Kind: store.MessageKindText, Text: "🔥"}
	label := c.buildBubbleText(singleMsg)
	if label == nil {
		t.Fatalf("expected label to be non-nil")
	}

	if !label.HasCSSClass("chat-bubble-single-emoji") {
		t.Errorf("expected label to have 'chat-bubble-single-emoji' class")
	}

	markup := label.Text()
	// Note: in GTK, label.Text() returns raw text, but we can verify the widget doesn't crash and has the CSS class
	if markup != "🔥" {
		t.Errorf("expected label text to be 🔥, got %q", markup)
	}

	normalMsg := store.StoredMessage{Kind: store.MessageKindText, Text: "Hello world"}
	normalLabel := c.buildBubbleText(normalMsg)
	if normalLabel == nil {
		t.Fatalf("expected normalLabel to be non-nil")
	}
	if normalLabel.HasCSSClass("chat-bubble-single-emoji") {
		t.Errorf("expected normal message NOT to have 'chat-bubble-single-emoji' class")
	}
}
