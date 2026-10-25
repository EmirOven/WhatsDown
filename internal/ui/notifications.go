package ui

import (
	"context"
	"fmt"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"os"
	"os/exec"
	"postbetasoftware.com/whatsdown/internal/avatar"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
)

var appInstance *gio.Application
var pendingNotificationChat string

func SetAppInstance(app *gio.Application) {
	appInstance = app
	if app == nil {
		return
	}
	action := notificationChatAction(func(chatJID string) {
		pendingNotificationChat = chatJID
		app.Activate()
		openPendingNotificationChat()
	})
	app.AddAction(action)
}

// dispatchDesktopNotification dispatches a system desktop notification.
func (c *MainController) dispatchDesktopNotification(chatJID, senderJID, title, body string) {
	glib.IdleAdd(func() {
		if appInstance != nil {
			notif := gio.NewNotification(title)
			notif.SetBody(body)
			notif.SetDefaultActionAndTarget("app.open-notification-chat", glib.NewVariantString(chatJID))
			notif.SetIcon(gio.NewThemedIcon("chat-message-new-symbolic"))
			avatar.LoadTexture(c.client, senderJID, false, func(_ *gdk.Texture) {
				if data, err := os.ReadFile(avatar.DiskPath(senderJID)); err == nil && len(data) > 0 {
					notif.SetIcon(gio.NewBytesIcon(glib.NewBytesWithGo(data)))
				}
				appInstance.SendNotification("", notif)
			})
			return
		}

		go func() {
			if _, err := exec.LookPath("notify-send"); err == nil {
				_ = exec.Command("notify-send",
					"-a", "WhatsDown",
					"-i", "chat-message-new-symbolic",
					title, body,
				).Run()
			}
		}()
	})
}

// formatNotificationContent extracts sender title and message preview for notifications.
func (c *MainController) formatNotificationContent(evt *events.Message) (string, string) {
	chatStr := evt.Info.Chat.ToNonAD().String()
	senderStr := evt.Info.Sender.ToNonAD().String()

	senderName := c.resolveContactName(senderStr)
	if senderName == "" || senderName == senderStr {
		if evt.Info.PushName != "" {
			senderName = evt.Info.PushName
		} else {
			senderName = cleanPhoneNumber(senderStr)
		}
	}
	if senderName == "" {
		senderName = "WhatsApp User"
	}

	var title string
	if evt.Info.IsGroup {
		chatName := c.resolveContactName(chatStr)
		if chatName == "" || chatName == chatStr {
			chatName = "Group"
		}
		title = fmt.Sprintf("%s (%s)", senderName, chatName)
	} else {
		title = senderName
	}

	body := "New message"
	if evt.Message != nil {
		kind, text, ok := store.DescribeMessage(evt.Message)
		if ok && text != "" {
			switch kind {
			case store.MessageKindPoll:
				body = "📊 Poll: " + text
			case store.MessageKindImage:
				body = "📷 Photo"
				if text != "" && text != "photo" {
					body += ": " + text
				}
			case store.MessageKindVideo:
				body = "🎥 Video"
				if text != "" && text != "video" {
					body += ": " + text
				}
			case store.MessageKindAudio, store.MessageKindVoice:
				body = "🎵 Audio message"
			case store.MessageKindDocument:
				body = "📄 " + text
			default:
				body = text
			}
		}
	}

	return title, body
}

// Register a typed app action so the desktop supplies the originating chat on activation.
func notificationChatAction(open func(string)) *gio.SimpleAction {
	action := gio.NewSimpleAction("open-notification-chat", glib.NewVariantType("s"))
	action.ConnectActivate(func(target *glib.Variant) {
		if target == nil {
			return
		}
		chat := target.String()
		if chat != "" {
			open(chat)
		}
	})
	return action
}

func openPendingNotificationChat() {
	c := activeMainWindow
	if pendingNotificationChat == "" || c == nil || c.store == nil {
		return
	}
	chat, err := c.store.GetChatOrContact(context.Background(), pendingNotificationChat)
	if err != nil {
		c.messageActionError(err)
		return
	}
	pendingNotificationChat = ""
	if c.mediaViewer != nil {
		c.mediaViewer.Close()
	}
	for remaining := c.window.Dialogs().NItems(); remaining > 0; remaining-- {
		if dialog := c.window.VisibleDialog(); dialog != nil {
			dialog.ForceClose()
		}
	}
	if c.sidebarStack != nil {
		c.sidebarStack.SetVisibleChildName("chats")
	}
	c.selectChat(chat)
	c.window.Present()
}
