package main

import (
	"os"
	"testing"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func TestBuildMainWindow(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("Skipping UI test: no display available")
	}

	gtk.Init()
	app := adw.NewApplication("com.test.whatsdown", 0)
	application = app

	app.ConnectActivate(func() {
		err := buildWindow(pageMain, MainUI)
		if err != nil {
			t.Fatalf("buildWindow failed: %v", err)
		}
		if activeMainWindow == nil {
			t.Fatalf("activeMainWindow is nil")
		}
		if activeMainWindow.sidebarStack == nil {
			t.Fatalf("sidebarStack is nil")
		}
		if activeMainWindow.sidebarStack.ChildByName("chats") == nil {
			t.Errorf("child 'chats' not found in sidebarStack")
		}
		if activeMainWindow.sidebarStack.ChildByName("archived") == nil {
			t.Errorf("child 'archived' not found in sidebarStack")
		}
		if activeMainWindow.sidebarStack.ChildByName("settings") == nil {
			t.Errorf("child 'settings' not found in sidebarStack")
		}

		// Test switching pages
		activeMainWindow.sidebarStack.SetVisibleChildName("archived")
		if activeMainWindow.sidebarStack.VisibleChildName() != "archived" {
			t.Errorf("expected visible child 'archived', got '%s'", activeMainWindow.sidebarStack.VisibleChildName())
		}
		activeMainWindow.sidebarStack.SetVisibleChildName("settings")
		if activeMainWindow.sidebarStack.VisibleChildName() != "settings" {
			t.Errorf("expected visible child 'settings', got '%s'", activeMainWindow.sidebarStack.VisibleChildName())
		}
		activeMainWindow.sidebarStack.SetVisibleChildName("chats")
		if activeMainWindow.sidebarStack.VisibleChildName() != "chats" {
			t.Errorf("expected visible child 'chats', got '%s'", activeMainWindow.sidebarStack.VisibleChildName())
		}

		// Test edit message flow
		activeMainWindow.startEditing(StoredMessage{
			ID:     "msg-test",
			Text:   "Hello edit",
			FromMe: true,
			Kind:   messageKindText,
		})

		activeMainWindow.cancelEditing()

		app.Quit()
	})

	app.Run(nil)
}
