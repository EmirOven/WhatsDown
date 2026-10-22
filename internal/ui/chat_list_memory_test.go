package ui

import (
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestChatListRebuildReleasesMenus(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no display available")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	gtk.Init()
	c := &MainController{}
	c.chatsList = gtk.NewListBox()
	c.archivedChatsList = gtk.NewListBox()
	chats := []store.Chat{{JID: "0@s.whatsapp.net", Name: "Test chat"}}
	c.renderChatList(chats, true)
	var released atomic.Int32
	const rebuilds = 10
	for i := 0; i < rebuilds; i++ {
		c.renderChatList(chats, false)
		// Every removed popover must finalize, even though its button callbacks
		// captured the popover and it owned the buttons through GTK.
		for row := range c.chatMenuCleanups {
			if coreglib.BaseObject(row.Parent()).Native() != coreglib.BaseObject(c.chatsList).Native() {
				continue
			}
			for child := row.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
				if popover, ok := child.(*gtk.Popover); ok {
					coreglib.WeakRefObject(popover, func() { released.Add(1) })
				}
			}
		}
	}
	c.renderChatList(nil, false)
	if len(c.chatMenuCleanups) != 1 {
		t.Fatalf("archived menu lost or old menus retained: %d", len(c.chatMenuCleanups))
	}
	ctx := glib.MainContextDefault()
	deadline := time.Now().Add(5 * time.Second)
	for released.Load() != rebuilds && time.Now().Before(deadline) {
		runtime.GC()
		for ctx.Iteration(false) {
		}
		time.Sleep(10 * time.Millisecond)
	}
	if released.Load() != rebuilds {
		t.Fatalf("released %d/%d removed menus", released.Load(), rebuilds)
	}
	c.renderChatList(nil, true)
	runtime.KeepAlive(c)
}

func TestSwitchChatReleasesMessageRows(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no display available")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	gtk.Init()
	c := &MainController{}
	c.messagesList = gtk.NewListBox()
	var released atomic.Int32
	const rows = 10
	for i := 0; i < rows; i++ {
		func() {
			msg := store.StoredMessage{ID: "test", ChatJID: "1@s.whatsapp.net", Kind: store.MessageKindText, Text: "Test message"}
			row := c.createMessageBubble(msg, false)
			var menus int
			var checkMenus func(gtk.Widgetter)
			checkMenus = func(widget gtk.Widgetter) {
				if popover, ok := widget.(*gtk.Popover); ok {
					menus++
					if popover.Child() != nil {
						t.Fatal("unopened message menu was populated while loading the chat")
					}
				}
				for child := gtk.BaseWidget(widget).FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
					checkMenus(child)
				}
			}
			checkMenus(row)
			if menus != 1 {
				t.Fatalf("found %d message menus, want 1", menus)
			}
			coreglib.WeakRefObject(row, func() { released.Add(1) })
			c.messagesList.Append(row)
		}()
		previousPreviews := c.messagePreviewContext
		c.loadMessages("another-chat")
		if previousPreviews != nil && previousPreviews.Err() == nil {
			t.Fatal("switching chats did not cancel the old chat's preview work")
		}
		if len(c.widgetSignalCleanups) != 0 {
			t.Fatalf("removed message retained %d callback owners", len(c.widgetSignalCleanups))
		}
	}
	ctx := glib.MainContextDefault()
	deadline := time.Now().Add(5 * time.Second)
	for released.Load() != rows && time.Now().Before(deadline) {
		runtime.GC()
		for ctx.Iteration(false) {
		}
		time.Sleep(10 * time.Millisecond)
	}
	if released.Load() != rows {
		t.Fatalf("released %d/%d removed message rows", released.Load(), rows)
	}
	runtime.KeepAlive(c)
}
