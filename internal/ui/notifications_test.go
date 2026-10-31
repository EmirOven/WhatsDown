package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"testing"
)

func TestNotificationActionRoutesToOriginatingChat(t *testing.T) {
	var opened []string
	var openedMsgs []string
	action := notificationChatAction(func(chat, msg string) {
		opened = append(opened, chat)
		openedMsgs = append(openedMsgs, msg)
	})
	for _, chat := range []string{"123@s.whatsapp.net", "456@g.us"} {
		action.Activate(glib.NewVariantString(chat))
		if len(opened) == 0 || opened[len(opened)-1] != chat {
			t.Fatalf("notification did not route to %s: %v", chat, opened)
		}
	}
	action.Activate(glib.NewVariantString("789@s.whatsapp.net|msg-abc-123"))
	if len(opened) != 3 || opened[2] != "789@s.whatsapp.net" || openedMsgs[2] != "msg-abc-123" {
		t.Fatalf("target with message ID failed: %v %v", opened, openedMsgs)
	}
	action.Activate(glib.NewVariantString(""))
	if len(opened) != 3 {
		t.Fatalf("empty target opened a chat: %v", opened)
	}
}

func TestNotificationAvatarUsesGroupInsteadOfSender(t *testing.T) {
	for _, test := range []struct{ chat, sender, want string }{
		{"123@g.us", "456@s.whatsapp.net", "123@g.us"},
		{"123@s.whatsapp.net", "456@s.whatsapp.net", "456@s.whatsapp.net"},
	} {
		if got := notificationAvatarJID(test.chat, test.sender); got != test.want {
			t.Fatalf("avatar=%s want %s", got, test.want)
		}
	}
}

func TestGroupNotificationPlaceholderCanBeExported(t *testing.T) {
	c := dialogTestController(t)
	c.window.Present()
	avatar := adw.NewAvatar(64, "Test Group", true)
	c.rootOverlay.AddOverlay(avatar)
	avatar.Allocate(64, 64, -1, nil)
	texture := gdk.BaseTexture(avatar.DrawToTexture(1))
	c.rootOverlay.RemoveOverlay(avatar)
	if texture.Width() != 64 || texture.Height() != 64 || texture.SaveToPNGBytes() == nil {
		t.Fatal("missing group placeholder image")
	}
}

func TestNotificationIDForChat(t *testing.T) {
	jid := "123456789@s.whatsapp.net"
	id := notificationIDForChat(jid)
	if id != "chat:"+jid {
		t.Fatalf("expected chat:%s, got %s", jid, id)
	}
}

func TestWithdrawChatNotification(t *testing.T) {
	c := dialogTestController(t)
	chatJID := "chat1@s.whatsapp.net"
	activeNotifsMu.Lock()
	activeNotifs[chatJID] = true
	activeNotifsMu.Unlock()

	c.withdrawChatNotification(chatJID)

	activeNotifsMu.Lock()
	active := activeNotifs[chatJID]
	activeNotifsMu.Unlock()
	if active {
		t.Fatal("expected notification to be withdrawn from active list")
	}
}

func TestMarkChatReadWithdrawsNotification(t *testing.T) {
	c := dialogTestController(t)
	chatJID := "testchat@s.whatsapp.net"
	activeNotifsMu.Lock()
	activeNotifs[chatJID] = true
	activeNotifsMu.Unlock()

	c.markChatRead(chatJID)

	activeNotifsMu.Lock()
	active := activeNotifs[chatJID]
	activeNotifsMu.Unlock()
	if active {
		t.Fatal("expected notification to be withdrawn on markChatRead")
	}
}

func TestOpenPendingNotificationChatBringsWindowToFrontAndOpensChat(t *testing.T) {
	c := dialogTestController(t)
	SetActiveMainWindow(c)
	t.Cleanup(func() { SetActiveMainWindow(nil) })

	chatJID := "target@s.whatsapp.net"
	pendingNotificationChat = chatJID
	activeNotifsMu.Lock()
	activeNotifs[chatJID] = true
	activeNotifsMu.Unlock()

	openPendingNotificationChat()

	if pendingNotificationChat != "" {
		t.Fatalf("expected pendingNotificationChat to be cleared, got %q", pendingNotificationChat)
	}
	if c.currentChat == nil || c.currentChat.JID != chatJID {
		t.Fatalf("expected currentChat to be %s, got %v", chatJID, c.currentChat)
	}
	activeNotifsMu.Lock()
	active := activeNotifs[chatJID]
	activeNotifsMu.Unlock()
	if active {
		t.Fatal("expected notification to be withdrawn after opening pending chat")
	}
}

func TestOpenPendingNotificationChatScrollsToMessage(t *testing.T) {
	c := dialogTestController(t)
	SetActiveMainWindow(c)
	t.Cleanup(func() { SetActiveMainWindow(nil) })

	chatJID := "scrolltarget@s.whatsapp.net"
	msgID := "msg-12345"
	pendingNotificationChat = chatJID
	pendingNotificationMsg = msgID

	openPendingNotificationChat()

	if pendingNotificationChat != "" || pendingNotificationMsg != "" {
		t.Fatalf("expected pending state cleared, got chat=%q msg=%q", pendingNotificationChat, pendingNotificationMsg)
	}
	if c.currentChat == nil || c.currentChat.JID != chatJID {
		t.Fatalf("expected currentChat to be %s, got %v", chatJID, c.currentChat)
	}
}

func TestWithdrawAllNotifications(t *testing.T) {
	c := dialogTestController(t)
	activeNotifsMu.Lock()
	activeNotifs["c1@s.whatsapp.net"] = true
	activeNotifs["c2@s.whatsapp.net"] = true
	activeNotifsMu.Unlock()

	c.withdrawAllNotifications()

	activeNotifsMu.Lock()
	count := len(activeNotifs)
	activeNotifsMu.Unlock()
	if count != 0 {
		t.Fatalf("expected 0 active notifications, got %d", count)
	}
}
