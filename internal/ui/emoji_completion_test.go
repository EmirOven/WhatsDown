package ui

import (
	"testing"
	"unicode/utf8"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func TestEmojiCompletionReplacesUnicodeQuery(t *testing.T) {
	for _, tc := range []struct{ name, text, before, want string }{
		{"ascii", "Hi :smile", "Hi :smile", "Hi 😃"},
		{"emoji before query", "😄 😄 :smile", "😄 😄 :smile", "😄 😄 😃"},
		{"accent before query", "café :smile", "café :smile", "café 😃"},
		{"joined emoji before query", "👨‍👩‍👧‍👦 :smile", "👨‍👩‍👧‍👦 :smile", "👨‍👩‍👧‍👦 😃"},
		{"closed shortcode", "😄 :smile:", "😄 :smile:", "😄 😃"},
		{"query in middle", "😄 :smile after", "😄 :smile", "😄 😃 after"},
		{"caret inside query", "😄 :smile after", "😄 :smi", "😄 😃 after"},
		{"caret inside closed query", "😄 :smile: after", "😄 :smi", "😄 😃 after"},
		{"unicode whitespace", "hi\u00a0:smile", "hi\u00a0:smile", "hi\u00a0😃"},
		{"punctuation remains", "Hi :smile.", "Hi :smile", "Hi 😃."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newFolderSidebar(t)
			c.messageEntry = gtk.NewTextView()
			c.emojiSuggestionBar = gtk.NewBox(gtk.OrientationHorizontal, 0)
			buffer := c.messageEntry.Buffer()
			buffer.SetEnableUndo(true)
			buffer.SetText(tc.text)
			buffer.PlaceCursor(buffer.IterAtOffset(utf8.RuneCountInString(tc.before)))
			c.emojiSuggestions = []string{"😃"}
			c.emojiSuggestionIndex = 0
			c.commitSelectedEmoji()
			if got := c.getMessageText(); got != tc.want {
				t.Fatalf("text=%q,want %q", got, tc.want)
			}
			if c.isEmojiSuggestionVisible() {
				t.Fatal("completion stayed visible")
			}
			buffer.Undo()
			if c.getMessageText() != tc.text {
				t.Fatal("emoji replacement was not one undo action")
			}
		})
	}
}
func TestStaleEmojiCompletionDoesNotInsertElsewhere(t *testing.T) {
	c := newFolderSidebar(t)
	c.messageEntry = gtk.NewTextView()
	c.emojiSuggestionBar = gtk.NewBox(gtk.OrientationHorizontal, 0)
	c.messageEntry.Buffer().SetText(":smile another word")
	c.messageEntry.Buffer().PlaceCursor(c.messageEntry.Buffer().EndIter())
	c.emojiSuggestions = []string{"😃"}
	c.commitSelectedEmoji()
	if c.getMessageText() != ":smile another word" {
		t.Fatal("stale completion changed unrelated text")
	}
}
func TestEmojiSuggestionsRecognizeUnicodeBoundary(t *testing.T) {
	c := newFolderSidebar(t)
	c.messageEntry = gtk.NewTextView()
	c.emojiSuggestionBar = gtk.NewBox(gtk.OrientationHorizontal, 0)
	c.messageEntry.Buffer().SetText("😄\u00a0:smile")
	c.messageEntry.Buffer().PlaceCursor(c.messageEntry.Buffer().EndIter())
	c.checkEmojiSuggestion()
	if !c.isEmojiSuggestionVisible() || c.emojiColonStartPos != 2 {
		t.Fatal("Unicode boundary was not detected")
	}
}
