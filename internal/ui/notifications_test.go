package ui

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"testing"
)

func TestNotificationActionRoutesToOriginatingChat(t *testing.T) {
	var opened []string
	action := notificationChatAction(func(chat string) { opened = append(opened, chat) })
	for _, chat := range []string{"123@s.whatsapp.net", "456@g.us"} {
		action.Activate(glib.NewVariantString(chat))
		if len(opened) == 0 || opened[len(opened)-1] != chat {
			t.Fatalf("notification did not route to %s: %v", chat, opened)
		}
	}
	action.Activate(glib.NewVariantString(""))
	if len(opened) != 2 {
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
