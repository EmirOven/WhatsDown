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
		if activeMainWindow.sidebarStack.ChildByName("calls") == nil {
			t.Errorf("child 'calls' not found in sidebarStack")
		}
		if activeMainWindow.sidebarStack.ChildByName("archived") == nil {
			t.Errorf("child 'archived' not found in sidebarStack")
		}
		if activeMainWindow.sidebarStack.ChildByName("settings") == nil {
			t.Errorf("child 'settings' not found in sidebarStack")
		}

		// Test switching pages
		activeMainWindow.sidebarStack.SetVisibleChildName("calls")
		if activeMainWindow.sidebarStack.VisibleChildName() != "calls" {
			t.Errorf("expected visible child 'calls', got '%s'", activeMainWindow.sidebarStack.VisibleChildName())
		}
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

		// Test voice recording UI flow
		if activeMainWindow.chatInputStack == nil {
			t.Fatalf("chatInputStack is nil")
		}
		if activeMainWindow.chatInputStack.VisibleChildName() != "normal" {
			t.Errorf("expected initial visible child 'normal', got '%s'", activeMainWindow.chatInputStack.VisibleChildName())
		}
		if activeMainWindow.voiceRecordBtn == nil {
			t.Fatalf("voiceRecordBtn is nil")
		}

		// Simulate starting recording on an active chat
		activeMainWindow.currentChat = &Chat{JID: "12345@s.whatsapp.net", Name: "Test User"}
		activeMainWindow.startVoiceRecording()
		if activeMainWindow.chatInputStack.VisibleChildName() != "recording" {
			t.Errorf("expected visible child 'recording' after start, got '%s'", activeMainWindow.chatInputStack.VisibleChildName())
		}
		if activeMainWindow.voicePauseBtn == nil || !activeMainWindow.voicePauseBtn.Visible() {
			t.Errorf("expected voicePauseBtn to be visible during recording")
		}
		if activeMainWindow.voiceStopBtn == nil || !activeMainWindow.voiceStopBtn.Visible() {
			t.Errorf("expected voiceStopBtn to be visible during recording")
		}
		if activeMainWindow.voiceSendBtn != nil && activeMainWindow.voiceSendBtn.Visible() {
			t.Errorf("expected voiceSendBtn to be hidden during recording")
		}

		// Pause & resume
		activeMainWindow.togglePauseVoiceRecording()
		if !activeMainWindow.voiceRecorder.Paused() {
			t.Errorf("expected voiceRecorder to be paused")
		}
		activeMainWindow.togglePauseVoiceRecording()
		if activeMainWindow.voiceRecorder.Paused() {
			t.Errorf("expected voiceRecorder to be resumed")
		}

		// Stop recording
		activeMainWindow.stopVoiceRecording()
		if activeMainWindow.voicePauseBtn.Visible() {
			t.Errorf("expected voicePauseBtn to be hidden when stopped")
		}
		if activeMainWindow.voiceStopBtn.Visible() {
			t.Errorf("expected voiceStopBtn to be hidden when stopped")
		}
		if !activeMainWindow.voiceSendBtn.Visible() {
			t.Errorf("expected voiceSendBtn to be visible when stopped")
		}

		// Delete recording
		activeMainWindow.deleteVoiceRecording()
		if activeMainWindow.chatInputStack.VisibleChildName() != "normal" {
			t.Errorf("expected visible child 'normal' after delete, got '%s'", activeMainWindow.chatInputStack.VisibleChildName())
		}

		// Test creating message bubbles for media, sticker, and text
		txtBubble := activeMainWindow.createMessageBubble(StoredMessage{
			ID:     "test-txt",
			Text:   "Hello world",
			FromMe: true,
			Kind:   messageKindText,
		})
		if txtBubble == nil {
			t.Errorf("expected text bubble to not be nil")
		}

		mediaBubble := activeMainWindow.createMessageBubble(StoredMessage{
			ID:     "test-media",
			Text:   "Media caption",
			FromMe: false,
			Kind:   messageKindImage,
		})
		if mediaBubble == nil {
			t.Errorf("expected media bubble to not be nil")
		}

		stickerBubble := activeMainWindow.createMessageBubble(StoredMessage{
			ID:     "test-sticker",
			FromMe: true,
			Kind:   messageKindSticker,
		})
		if stickerBubble == nil {
			t.Errorf("expected sticker bubble to not be nil")
		}

		// Test clipboard media paste handling
		activeMainWindow.currentChat = nil
		if activeMainWindow.handleClipboardMediaPaste() {
			t.Errorf("expected handleClipboardMediaPaste to return false when no chat is active")
		}
		activeMainWindow.currentChat = &Chat{
			JID:  "12345@s.whatsapp.net",
			Name: "Test Chat",
		}
		// With empty/text-only clipboard, should return false so normal text paste works
		if activeMainWindow.handleClipboardMediaPaste() {
			t.Errorf("expected handleClipboardMediaPaste to return false for empty/text clipboard")
		}

		app.Quit()
	})

	app.Run(nil)
}

func TestParseFilePathFromText(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "whatsdown_test_*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// 1. Direct path
	if path, ok := parseFilePathFromText(tmpPath); !ok || path != tmpPath {
		t.Errorf("expected direct path %s, got %s (ok=%v)", tmpPath, path, ok)
	}

	// 2. file:// URI
	fileURI := "file://" + tmpPath
	if path, ok := parseFilePathFromText(fileURI); !ok || path != tmpPath {
		t.Errorf("expected file URI path %s, got %s (ok=%v)", tmpPath, path, ok)
	}

	// 3. GNOME copied files format (copy\nfile://...)
	gnomeFormat := "copy\n" + fileURI + "\r\n"
	if path, ok := parseFilePathFromText(gnomeFormat); !ok || path != tmpPath {
		t.Errorf("expected GNOME format path %s, got %s (ok=%v)", tmpPath, path, ok)
	}

	// 4. Quoted path
	quoted := `"` + tmpPath + `"`
	if path, ok := parseFilePathFromText(quoted); !ok || path != tmpPath {
		t.Errorf("expected quoted path %s, got %s (ok=%v)", tmpPath, path, ok)
	}

	// 5. Non-file regular text message
	if path, ok := parseFilePathFromText("Hello, this is a normal WhatsApp text message!"); ok {
		t.Errorf("expected regular message to return ok=false, got path=%s", path)
	}

	// 6. Non-existent file path
	if path, ok := parseFilePathFromText("/does/not/exist/image_never_exists.png"); ok {
		t.Errorf("expected non-existent file to return ok=false, got path=%s", path)
	}

	// 7. Directory should return false (only files can be sent as media/attachments)
	if path, ok := parseFilePathFromText(os.TempDir()); ok {
		t.Errorf("expected directory to return ok=false, got path=%s", path)
	}
}
