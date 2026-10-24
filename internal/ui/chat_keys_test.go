package ui

import (
	"context"
	"testing"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func TestSearchChildFocusDoesNotRedirectTyping(t *testing.T) {
	c := newFolderSidebar(t)
	adw.Init()
	app := gtk.NewApplication("com.postbetasoftware.WhatsDown.FocusTest", 0)
	if err := app.Register(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.window = adw.NewApplicationWindow(app)
	t.Cleanup(c.window.Destroy)
	box := gtk.NewBox(gtk.OrientationVertical, 4)
	c.chatsSearchEntry = gtk.NewSearchEntry()
	c.messageEntry = gtk.NewTextView()
	box.Append(c.chatsSearchEntry)
	box.Append(c.messageEntry)
	c.window.SetContent(box)
	c.window.Present()
	c.chatsSearchEntry.GrabFocus()
	pumpSidebarEvents(20 * time.Millisecond)
	if !c.isAnySearchEntryFocused() {
		t.Fatal("search entry child focus was not detected")
	}
	if c.handleChatViewKeyPress(gdk.KEY_a, 0, 0) {
		t.Fatal("search keystroke was consumed by message composer")
	}
	if c.getMessageText() != "" {
		t.Fatal("search keystroke appeared in composer")
	}
}
