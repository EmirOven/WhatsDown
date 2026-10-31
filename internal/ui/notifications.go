package ui

import (
	"context"
	"fmt"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"os/exec"
	"postbetasoftware.com/whatsdown/internal/avatar"
	"strings"
	"sync"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/store"
)

var appInstance *gio.Application
var pendingNotificationChat string
var pendingNotificationMsg string

var (
	activeNotifsMu sync.Mutex
	activeNotifs   = make(map[string]bool)
)

func notificationIDForChat(chatJID string) string {
	return "chat:" + chatJID
}

func withdrawNotificationForJID(jid string) {
	if jid == "" {
		return
	}
	activeNotifsMu.Lock()
	delete(activeNotifs, jid)
	activeNotifsMu.Unlock()

	if appInstance != nil {
		appInstance.WithdrawNotification(notificationIDForChat(jid))
	}
}

func (c *MainController) withdrawChatNotification(chatJID string) {
	if chatJID == "" {
		return
	}
	withdrawNotificationForJID(chatJID)
	if c != nil && c.store != nil {
		if pn := c.store.ResolveLIDtoPN(context.Background(), chatJID); pn != "" && pn != chatJID {
			withdrawNotificationForJID(pn)
		}
	}
}

// WithdrawChatNotification withdraws the notification for the specified chat JID.
func WithdrawChatNotification(chatJID string) {
	if c := activeMainWindow; c != nil {
		c.withdrawChatNotification(chatJID)
	} else {
		withdrawNotificationForJID(chatJID)
	}
}

func (c *MainController) withdrawAllNotifications() {
	activeNotifsMu.Lock()
	var toWithdraw []string
	for jid := range activeNotifs {
		toWithdraw = append(toWithdraw, jid)
	}
	activeNotifs = make(map[string]bool)
	activeNotifsMu.Unlock()

	if appInstance != nil {
		for _, jid := range toWithdraw {
			appInstance.WithdrawNotification(notificationIDForChat(jid))
		}
		if c != nil {
			for _, ch := range c.chats {
				appInstance.WithdrawNotification(notificationIDForChat(ch.JID))
			}
			for _, ch := range c.archivedChats {
				appInstance.WithdrawNotification(notificationIDForChat(ch.JID))
			}
		}
	}
}

func (c *MainController) isWindowFocused() bool {
	if c == nil || c.window == nil {
		return false
	}
	return c.window.IsActive()
}

func (c *MainController) isCurrentChatAndFocused(chatJID string) bool {
	if c == nil || c.window == nil || c.currentChat == nil {
		return false
	}
	return c.window.IsActive() && c.isSameChat(c.currentChat.JID, chatJID)
}

// BringToFront raises and focuses the application window above all other windows.
func (c *MainController) BringToFront() {
	if c == nil || c.window == nil {
		return
	}
	c.window.SetVisible(true)
	c.window.Unminimize()
	c.window.PresentWithTime(gdk.CURRENT_TIME)
	c.window.Present()
	if c.messageEntry != nil {
		c.messageEntry.GrabFocus()
	}

	glib.IdleAdd(func() {
		if c == nil || c.window == nil {
			return
		}
		c.window.SetVisible(true)
		c.window.Unminimize()
		c.window.PresentWithTime(gdk.CURRENT_TIME)
		c.window.Present()
		if c.messageEntry != nil {
			c.messageEntry.GrabFocus()
		}
	})
}

func SetAppInstance(app *gio.Application) {
	appInstance = app
	if app == nil {
		return
	}
	action := notificationChatAction(func(chatJID, msgID string) {
		pendingNotificationChat = chatJID
		pendingNotificationMsg = msgID
		app.Activate()
		openPendingNotificationChat()
	})
	app.AddAction(action)
}

func playNotificationSound() {
	go func() {
		if _, err := exec.LookPath("canberra-gtk-play"); err == nil {
			if err := exec.Command("canberra-gtk-play", "-i", "message-new-instant", "-d", "WhatsDown").Run(); err == nil {
				return
			}
		}
		soundFile := "/usr/share/sounds/freedesktop/stereo/message-new-instant.oga"
		if _, err := exec.LookPath("pw-play"); err == nil {
			if err := exec.Command("pw-play", soundFile).Run(); err == nil {
				return
			}
		}
		if _, err := exec.LookPath("paplay"); err == nil {
			_ = exec.Command("paplay", soundFile).Run()
		}
	}()
}

// dispatchDesktopNotification dispatches a system desktop notification.
func (c *MainController) dispatchDesktopNotification(chatJID, senderJID, msgID, title, body string) {
	playNotificationSound()
	glib.IdleAdd(func() {
		if appInstance != nil {
			notif := gio.NewNotification(title)
			notif.SetBody(body)
			target := chatJID
			if msgID != "" {
				target = chatJID + "|" + msgID
			}
			notif.SetDefaultActionAndTarget("app.open-notification-chat", glib.NewVariantString(target))
			notif.SetIcon(gio.NewThemedIcon("chat-message-new-symbolic"))
			avatarJID := notificationAvatarJID(chatJID, senderJID)
			notifID := notificationIDForChat(chatJID)

			activeNotifsMu.Lock()
			activeNotifs[chatJID] = true
			activeNotifsMu.Unlock()

			avatar.LoadTexture(c.client, avatarJID, false, func(texture *gdk.Texture) {
				activeNotifsMu.Lock()
				stillActive := activeNotifs[chatJID]
				activeNotifsMu.Unlock()
				if !stillActive {
					return
				}
				if c != nil && c.isCurrentChatAndFocused(chatJID) {
					withdrawNotificationForJID(chatJID)
					return
				}

				if texture != nil {
					notif.SetIcon(gio.NewBytesIcon(texture.SaveToPNGBytes()))
				} else if strings.HasSuffix(chatJID, "@g.us") {
					name := c.resolveContactName(chatJID)
					if name == "" || name == chatJID {
						name = "Group"
					}
					placeholder := adw.NewAvatar(64, name, true)
					c.rootOverlay.AddOverlay(placeholder)
					placeholder.Allocate(64, 64, -1, nil)
					texture := gdk.BaseTexture(placeholder.DrawToTexture(1))
					c.rootOverlay.RemoveOverlay(placeholder)
					notif.SetIcon(gio.NewBytesIcon(texture.SaveToPNGBytes()))
				}
				appInstance.SendNotification(notifID, notif)
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
func notificationChatAction(open func(chatJID, msgID string)) *gio.SimpleAction {
	action := gio.NewSimpleAction("open-notification-chat", glib.NewVariantType("s"))
	action.ConnectActivate(func(target *glib.Variant) {
		if target == nil {
			return
		}
		raw := strings.Trim(strings.TrimSpace(target.String()), "'\"")
		if raw == "" {
			return
		}
		parts := strings.SplitN(raw, "|", 2)
		chatJID := parts[0]
		msgID := ""
		if len(parts) > 1 {
			msgID = parts[1]
		}
		open(chatJID, msgID)
	})
	return action
}

func openPendingNotificationChat() {
	c := activeMainWindow
	if c == nil {
		return
	}
	targetChat := pendingNotificationChat
	targetMsg := pendingNotificationMsg
	pendingNotificationChat = ""
	pendingNotificationMsg = ""

	if c.mediaViewer != nil {
		c.mediaViewer.Close()
	}
	if c.window != nil {
		for remaining := c.window.Dialogs().NItems(); remaining > 0; remaining-- {
			if dialog := c.window.VisibleDialog(); dialog != nil {
				dialog.ForceClose()
			}
		}
	}

	if targetChat != "" {
		var chat store.Chat
		var err error
		if c.store != nil {
			chat, err = c.store.GetChatOrContact(context.Background(), targetChat)
		}
		if err != nil || chat.JID == "" {
			chat = store.Chat{JID: targetChat}
		}

		if c.sidebarStack != nil {
			if chat.Archived {
				c.sidebarStack.SetVisibleChildName("archived")
			} else {
				c.sidebarStack.SetVisibleChildName("chats")
			}
		}
		c.selectChat(chat)
		if targetMsg != "" {
			c.jumpToMessage(targetChat, targetMsg)
		}
		c.withdrawChatNotification(targetChat)
	}

	c.BringToFront()
}

func notificationAvatarJID(chatJID, senderJID string) string {
	if strings.HasSuffix(chatJID, "@g.us") {
		return chatJID
	}
	return senderJID
}
