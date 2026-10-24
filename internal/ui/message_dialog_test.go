package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func dialogTestController(t *testing.T) *MainController {
	c := newFolderSidebar(t)
	adw.Init()
	app := gtk.NewApplication("com.postbetasoftware.WhatsDown.DialogTest."+strings.ReplaceAll(t.Name(), "/", "."), 0)
	if err := app.Register(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.window = adw.NewApplicationWindow(app)
	c.rootOverlay = gtk.NewOverlay()
	c.window.SetContent(c.rootOverlay)
	t.Cleanup(c.window.Destroy)
	return c
}

func walkDialog(widget gtk.Widgetter, visit func(gtk.Widgetter)) {
	visit(widget)
	for child := gtk.BaseWidget(widget).FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
		walkDialog(child, visit)
	}
}

func TestForwardDialogUsesIdentityRowsAndHeaderActions(t *testing.T) {
	c := dialogTestController(t)
	c.showForwardMessages([]store.StoredMessage{{Kind: store.MessageKindText, Text: "forward me"}})
	var win *gtk.Window
	for _, item := range gtk.WindowListToplevels() {
		if candidate, ok := item.(*gtk.Window); ok && candidate.Title() == "Forward Messages" {
			win = candidate
			break
		}
	}
	if win == nil {
		t.Fatal("forward dialog did not open")
	}
	t.Cleanup(win.Destroy)
	var list *gtk.ListBox
	var search *gtk.SearchEntry
	var forward, cancel *gtk.Button
	walkDialog(win, func(widget gtk.Widgetter) {
		switch w := widget.(type) {
		case *gtk.ListBox:
			list = w
		case *gtk.SearchEntry:
			search = w
		case *gtk.Button:
			if w.Label() == "Forward" {
				forward = w
			}
			if w.Label() == "Cancel" {
				cancel = w
			}
		}
	})
	if list == nil || search == nil || forward == nil || cancel == nil {
		t.Fatal("dialog controls missing")
	}
	if forward.Sensitive() {
		t.Fatal("forward enabled without destination")
	}
	if !forward.IsAncestor(win.Titlebar()) || !cancel.IsAncestor(win.Titlebar()) {
		t.Fatal("dialog actions are not in header bar")
	}
	row := list.RowAtIndex(0)
	var avatarFound bool
	walkDialog(row, func(widget gtk.Widgetter) {
		if _, ok := widget.(*adw.Avatar); ok {
			avatarFound = true
		}
	})
	if !avatarFound {
		t.Fatal("destination row has no avatar")
	}
	list.SelectRow(row)
	if !forward.Sensitive() {
		t.Fatal("selecting a destination did not enable forward")
	}
	search.SetText("no matching person")
	pumpSidebarEvents(300 * time.Millisecond)
	if list.RowAtIndex(0) != nil || forward.Sensitive() {
		t.Fatal("search retained stale destination selection")
	}
}

func TestMessageInfoAutomaticallyRefreshes(t *testing.T) {
	c := dialogTestController(t)
	ctx := context.Background()
	msg := store.StoredMessage{ChatJID: "0@a.test", ID: "info-refresh", FromMe: true, Kind: store.MessageKindText, Text: "Hello", Timestamp: time.Now()}
	if err := c.store.SaveMessage(ctx, msg); err != nil {
		t.Fatal(err)
	}
	if err := c.store.SetMessageRecipients(ctx, msg.ChatJID, msg.ID, []string{"0@b.test"}); err != nil {
		t.Fatal(err)
	}
	view := c.buildMessageInfo(msg)
	t.Cleanup(view.window.Destroy)
	view.window.Present()
	hasGroup := func(title string) bool {
		found := false
		walkDialog(view.window, func(widget gtk.Widgetter) {
			if group, ok := widget.(*adw.PreferencesGroup); ok && group.Title() == title {
				found = true
			}
		})
		return found
	}
	if !hasGroup("Pending Delivery · 1") {
		t.Fatal("pending recipient missing")
	}
	if err := c.store.SaveMessageReceipt(ctx, msg.ChatJID, []string{msg.ID}, "0@b.test", store.MessageStatusRead, time.Now()); err != nil {
		t.Fatal(err)
	}
	pumpSidebarEvents(1200 * time.Millisecond)
	if !hasGroup("Read by · 1") || !hasGroup("Pending Delivery · 0") {
		t.Fatal("open message info did not automatically refresh")
	}
	view.window.Close()
	pumpSidebarEvents(20 * time.Millisecond)
}
