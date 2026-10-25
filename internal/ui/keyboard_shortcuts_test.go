package ui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func shortcutComposer(t *testing.T, text string) *MainController {
	c := dialogTestController(t)
	c.currentChat = &store.Chat{JID: "0@a.test"}
	c.messageEntry = gtk.NewTextView()
	c.rootOverlay.SetChild(c.messageEntry)
	c.messageEntry.Buffer().SetText(text)
	c.window.SetDefaultSize(400, 200)
	c.window.Present()
	c.messageEntry.GrabFocus()
	pumpSidebarEvents(20 * time.Millisecond)
	return c
}
func selectComposerText(c *MainController, text string) {
	all := c.getMessageText()
	offset := strings.Index(all, text)
	start := utf8.RuneCountInString(all[:offset])
	buffer := c.messageEntry.Buffer()
	buffer.SelectRange(buffer.IterAtOffset(start), buffer.IterAtOffset(start+utf8.RuneCountInString(text)))
}
func TestSelectedTextShortcuts(t *testing.T) {
	for _, tc := range []struct {
		name, input, selected, want string
		key                         uint
	}{
		{"bold unicode", "before αβ after", "αβ", "before *αβ* after", gdk.KEY_b},
		{"italic", "hello world", "world", "hello _world_", gdk.KEY_i},
		{"strike", "hello world", "world", "hello ~world~", gdk.KEY_s},
		{"monospace", "before a\nb after", "a\nb", "before ```a\nb``` after", gdk.KEY_m},
		{"multiline", " one \n two ", " one \n two ", " *one* \n *two* ", gdk.KEY_b},
		{"quote", "one\ntwo", "one\ntwo", "> one\n> two", gdk.KEY_q},
		{"partial quote", "before selected after", "selected", "before \n> selected\n after", gdk.KEY_q},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := shortcutComposer(t, tc.input)
			selectComposerText(c, tc.selected)
			if !c.handleChatViewKeyPress(tc.key, 0, gdk.ControlMask) {
				t.Fatal("shortcut not handled")
			}
			if got := c.getMessageText(); got != tc.want {
				t.Fatalf("text=%q,want %q", got, tc.want)
			}
			if _, _, ok := c.messageEntry.Buffer().SelectionBounds(); !ok {
				t.Fatal("formatting discarded selection")
			}
		})
	}
}
func TestFormattingRequiresComposerSelection(t *testing.T) {
	c := shortcutComposer(t, "unchanged")
	for _, key := range []uint{gdk.KEY_b, gdk.KEY_i, gdk.KEY_s, gdk.KEY_q, gdk.KEY_m} {
		c.handleChatViewKeyPress(key, 0, gdk.ControlMask)
		if c.getMessageText() != "unchanged" {
			t.Fatal("formatted without selection")
		}
	}
	if c.handleChatViewKeyPress(gdk.KEY_u, 0, gdk.ControlMask) {
		t.Fatal("underline shortcut was not omitted")
	}
	c.messageEntry.Buffer().SelectRange(c.messageEntry.Buffer().StartIter(), c.messageEntry.Buffer().EndIter())
	box := gtk.NewBox(gtk.OrientationVertical, 0)
	c.rootOverlay.SetChild(nil)
	box.Append(c.messageEntry)
	c.chatsSearchEntry = gtk.NewSearchEntry()
	box.Append(c.chatsSearchEntry)
	c.rootOverlay.SetChild(box)
	c.chatsSearchEntry.GrabFocus()
	pumpSidebarEvents(10 * time.Millisecond)
	if c.handleChatViewKeyPress(gdk.KEY_b, 0, gdk.ControlMask) || c.getMessageText() != "unchanged" {
		t.Fatal("formatted text from another focused widget")
	}
}
func TestFormattingTogglesWithOneSelection(t *testing.T) {
	c := shortcutComposer(t, "word")
	selectComposerText(c, "word")
	c.handleChatViewKeyPress(gdk.KEY_b, 0, gdk.ControlMask)
	c.handleChatViewKeyPress(gdk.KEY_b, 0, gdk.ControlMask)
	if c.getMessageText() != "word" {
		t.Fatal("bold did not toggle off")
	}
}
func TestUpEditsLastSentTextWithoutOverwritingDraft(t *testing.T) {
	c := shortcutComposer(t, "draft")
	if err := c.store.SaveMessage(t.Context(), store.StoredMessage{ChatJID: c.currentChat.JID, ID: "mine", FromMe: true, Kind: store.MessageKindText, Text: "last sent", Timestamp: time.Unix(100, 0)}); err != nil {
		t.Fatal(err)
	}
	if err := c.store.SaveMessage(t.Context(), store.StoredMessage{ChatJID: c.currentChat.JID, ID: "incoming", Kind: store.MessageKindText, Text: "reply", Timestamp: time.Unix(200, 0)}); err != nil {
		t.Fatal(err)
	}
	if c.handleEntryKeyPress(gdk.KEY_Up, 0, 0) || c.editingMessage != nil || c.getMessageText() != "draft" {
		t.Fatal("Up overwrote a draft")
	}
	c.setMessageText("")
	if !c.handleEntryKeyPress(gdk.KEY_Up, 0, 0) || c.editingMessage == nil || c.editingMessage.ID != "mine" || c.getMessageText() != "last sent" {
		t.Fatal("Up did not edit last outgoing text")
	}
}
func TestChatShortcutOrderSkipsHeadersAndWraps(t *testing.T) {
	c := newFolderSidebar(t)
	if _, err := c.store.CreateChatFolder(t.Context(), "Folder", "#3584e4", []string{"0@a.test"}); err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	c.currentChat = &store.Chat{JID: "0@b.test"}
	next, ok := c.adjacentVisibleChat(1)
	if !ok || next.JID != "0@c.test" {
		t.Fatalf("next=%+v", next)
	}
	prev, ok := c.adjacentVisibleChat(-1)
	if !ok || prev.JID != "0@a.test" {
		t.Fatalf("previous=%+v", prev)
	}
	c.currentChat = &store.Chat{JID: "0@a.test"}
	prev, ok = c.adjacentVisibleChat(-1)
	if !ok || prev.JID != "0@d.test" {
		t.Fatalf("wrapped previous=%+v", prev)
	}
}
func TestShortcutHelpIsNativeModal(t *testing.T) {
	c := shortcutComposer(t, "")
	if !c.handleChatViewKeyPress(gdk.KEY_slash, 0, gdk.ControlMask) {
		t.Fatal("help shortcut not handled")
	}
	dialog := c.window.VisibleDialog()
	if dialog == nil || dialog.Title() != "Keyboard Shortcuts" {
		t.Fatal("help dialog not opened")
	}
	defer dialog.Close()
	labels := 0
	walkDialog(dialog, func(w gtk.Widgetter) {
		if label, ok := w.(*gtk.Label); ok && label.HasCSSClass("shortcut-keycap") {
			labels++
		}
	})
	if labels != 21 {
		t.Fatalf("shortcut labels=%d", labels)
	}
	if c.handleChatViewKeyPress(gdk.KEY_Tab, 0, gdk.ControlMask) {
		t.Fatal("chat navigation intercepted modal dialog input")
	}
}

func TestUpEditsWithoutComposerFocus(t *testing.T) {
	c := shortcutComposer(t, "")
	if err := c.store.SaveMessage(t.Context(), store.StoredMessage{ChatJID: c.currentChat.JID, ID: "last", FromMe: true, Kind: store.MessageKindText, Text: "edit me", Timestamp: time.Unix(100, 0)}); err != nil {
		t.Fatal(err)
	}
	box := gtk.NewBox(gtk.OrientationVertical, 0)
	c.rootOverlay.SetChild(nil)
	box.Append(c.messageEntry)
	button := gtk.NewButtonWithLabel("Other control")
	box.Append(button)
	c.rootOverlay.SetChild(box)
	button.GrabFocus()
	pumpSidebarEvents(20 * time.Millisecond)
	if c.composerFocused() {
		t.Fatal("test still has composer focus")
	}
	if !c.handleChatViewKeyPress(gdk.KEY_Up, 0, 0) || c.editingMessage == nil || c.getMessageText() != "edit me" {
		t.Fatal("global Up did not edit last message")
	}
}
