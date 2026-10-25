package ui

import (
	"os"
	"testing"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestReplyKeepsScrollPosition(t *testing.T) {
	c := newFolderSidebar(t)
	c.messagesScrolledWindow = gtk.NewScrolledWindow()
	adjustment := c.messagesScrolledWindow.VAdjustment()
	adjustment.SetUpper(1000)
	adjustment.SetPageSize(100)
	adjustment.SetValue(350)
	c.startReply(store.StoredMessage{ID: "reply", FromMe: true})
	pumpSidebarEvents(30 * time.Millisecond)
	if adjustment.Value() != 350 || c.autoScrollBottom {
		t.Fatal("selecting a reply scrolled the conversation")
	}
}
func TestVoiceControlsDoNotTriggerBubbleReply(t *testing.T) {
	c := newFolderSidebar(t)
	box := c.buildAudioWidget(store.StoredMessage{Kind: store.MessageKindVoice})
	t.Cleanup(func() { c.releaseMessageWidget(box) })
	buttons, scales := 0, 0
	walkDialog(box, func(w gtk.Widgetter) {
		switch v := w.(type) {
		case *gtk.Button:
			buttons++
			if messageTargetAllowsReply(v) {
				t.Fatal("voice button triggers double-click reply")
			}
			if v.VAlign() != gtk.AlignCenter {
				t.Fatal("voice button stretches vertically")
			}
		case *gtk.Scale:
			scales++
			if messageTargetAllowsReply(v) {
				t.Fatal("scrubber triggers double-click reply")
			}
		}
	})
	if buttons != 2 || scales != 1 {
		t.Fatalf("voice controls=%d buttons,%d scales", buttons, scales)
	}
	if !messageTargetAllowsReply(gtk.NewLabel("message text")) {
		t.Fatal("plain messages no longer support double-click reply")
	}
}
func TestSettingsAboutRowOpensNativeAboutDialog(t *testing.T) {
	c := dialogTestController(t)
	xml, err := os.ReadFile("../../ui/components/SettingsMenu.ui")
	if err != nil {
		t.Fatal(err)
	}
	b := gtk.NewBuilder()
	if err := b.AddFromString(string(xml)); err != nil {
		t.Fatal(err)
	}
	row, ok := castObject[*adw.ActionRow](b, "settings_about_row")
	if !ok || !row.Activatable() {
		t.Fatal("About row cannot be activated")
	}
	c.bindSettingsWidgets(b)
	c.window.Present()
	row.Emit("activated")
	pumpSidebarEvents(40 * time.Millisecond)
	dialog := c.window.VisibleDialog()
	if dialog == nil {
		t.Fatal("About row did not open a native dialog")
	}
	// The native object must be AdwAboutDialog rather than custom settings content.
	if dialog.TypeFromInstance().Name() != "AdwAboutDialog" {
		t.Fatalf("wrong dialog type %s", dialog.TypeFromInstance().Name())
	}
	dialog.Close()
}

func TestSettingsButtonOpensShortcutHelp(t *testing.T) {
	c := dialogTestController(t)
	xml, err := os.ReadFile("../../ui/components/SettingsMenu.ui")
	if err != nil {
		t.Fatal(err)
	}
	builder := gtk.NewBuilder()
	if err := builder.AddFromString(string(xml)); err != nil {
		t.Fatal(err)
	}
	button, ok := castObject[*gtk.Button](builder, "settings_shortcuts_btn")
	if !ok {
		t.Fatal("shortcut help button missing")
	}
	c.bindSettingsWidgets(builder)
	c.window.Present()
	button.Emit("clicked")
	dialog := c.window.VisibleDialog()
	if dialog == nil || dialog.Title() != "Keyboard Shortcuts" {
		t.Fatal("Settings did not open shortcut help")
	}
	defer dialog.Close()
	caps, legacy := 0, 0
	walkDialog(dialog, func(w gtk.Widgetter) {
		if label, ok := w.(*gtk.Label); ok && label.HasCSSClass("shortcut-keycap") {
			caps++
		}
		if _, ok := w.(*gtk.ShortcutLabel); ok {
			legacy++
		}
	})
	if caps != 21 || legacy != 0 {
		t.Fatalf("keycaps=%d,legacy shortcut labels=%d", caps, legacy)
	}
}
