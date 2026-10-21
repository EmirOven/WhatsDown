package ui

import (
	"fmt"
	"os/exec"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
)

var appInstance *gio.Application

func SetAppInstance(app *gio.Application) {
	appInstance = app
}

// dispatchDesktopNotification dispatches a system desktop notification.
func dispatchDesktopNotification(title, body string) {
	glib.IdleAdd(func() {
		if appInstance != nil {
			notif := gio.NewNotification(title)
			notif.SetBody(body)
			notif.SetIcon(gio.NewThemedIcon("chat-message-new-symbolic"))
			appInstance.SendNotification("", notif)
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
