package ui

import (
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
