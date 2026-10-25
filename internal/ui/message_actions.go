package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

func (c *MainController) messageActionError(err error) {
	alert := adw.NewAlertDialog("Message action failed", err.Error())
	alert.AddResponse("close", "Close")
	alert.Present(c.window)
}

func (c *MainController) appendMessageActions(box *gtk.Box, pop *gtk.Popover, msg store.StoredMessage) {
	add := func(icon, label string, action func()) {
		button := newMenuButtonContent(icon, label)
		c.trackWidgetSignal(button, button, button.ConnectClicked(func() { pop.Popdown(); action() }))
		box.Append(button)
	}
	if msg.FromMe {
		add("dialog-information-symbolic", "Message info…", func() { c.showMessageInfo(msg) })
	}
	if msg.Kind != store.MessageKindDeleted {
		add("mail-forward-symbolic", "Forward…", func() { c.showForwardMessages([]store.StoredMessage{msg}) })
	}
	add("object-select-symbolic", "Select messages…", func() { c.showMessageSelection(msg.ID) })
	add("user-trash-symbolic", "Delete…", func() { c.confirmDeleteMessages([]store.StoredMessage{msg}) })
}

func (c *MainController) confirmDeleteMessages(messages []store.StoredMessage) {
	if len(messages) == 0 {
		return
	}
	alert := adw.NewAlertDialog("Delete messages?", fmt.Sprintf("Delete %d selected message(s)? Delete for me removes them from this device.", len(messages)))
	alert.AddResponse("cancel", "Cancel")
	alert.AddResponse("me", "Delete for me")
	allMine := true
	for _, msg := range messages {
		allMine = allMine && msg.FromMe && msg.Kind != store.MessageKindDeleted
	}
	if allMine {
		alert.AddResponse("everyone", "Delete for everyone")
	}
	alert.SetDefaultResponse("cancel")
	alert.SetCloseResponse("cancel")
	alert.ConnectResponse(func(response string) {
		if response != "me" && response != "everyone" {
			return
		}
		go func() {
			for _, msg := range messages {
				if err := c.deleteSelectedMessage(msg, response == "everyone"); err != nil {
					glib.IdleAdd(func() { c.messageActionError(err); c.refreshMessageActions(messages[0].ChatJID) })
					return
				}
			}
			glib.IdleAdd(func() { c.refreshMessageActions(messages[0].ChatJID) })
		}()
	})
	alert.Present(c.window)
}

func (c *MainController) refreshMessageActions(chat string) {
	if c.currentChat != nil && c.isSameChat(c.currentChat.JID, chat) {
		c.cancelMessageSelection()
		c.loadMessages(c.currentChat.JID)
	}
	c.loadChats()
}

func (c *MainController) deleteSelectedMessage(msg store.StoredMessage, everyone bool) error {
	if c.store == nil {
		return fmt.Errorf("message store is unavailable")
	}
	if everyone {
		if !msg.FromMe {
			return fmt.Errorf("only your own messages can be deleted for everyone")
		}
		if !c.store.IsDemo() {
			if c.client == nil || !c.client.IsConnected() {
				return fmt.Errorf("connect to WhatsApp to delete for everyone")
			}
			jid, err := types.ParseJID(msg.ChatJID)
			if err != nil {
				return err
			}
			if _, err = c.client.SendMessage(context.Background(), jid, c.client.BuildRevoke(jid, types.EmptyJID, msg.ID)); err != nil {
				return err
			}
		}
	}
	return c.store.DeleteMessage(context.Background(), msg.ChatJID, msg.ID, everyone)
}

func (c *MainController) showForwardMessages(messages []store.StoredMessage) {
	if c.store == nil || len(messages) == 0 {
		return
	}
	for _, msg := range messages {
		if _, err := buildForwardMessage(msg); err != nil {
			c.messageActionError(err)
			return
		}
	}
	chats, err := c.store.ListChats(context.Background())
	if err != nil {
		c.messageActionError(err)
		return
	}
	archived, err := c.store.ListArchivedChats(context.Background())
	if err != nil {
		c.messageActionError(err)
		return
	}
	chats = append(chats, archived...)
	win, box, header := c.newMessageDialog("Forward Messages", "Cancel")
	title := adw.NewWindowTitle("Forward Messages", "")
	header.SetTitleWidget(title)
	forward := gtk.NewButtonWithLabel("Forward")
	forward.AddCSSClass("suggested-action")
	forward.SetSensitive(false)
	header.PackEnd(forward)
	win.SetDefaultWidget(forward)
	search := gtk.NewSearchEntry()
	search.SetPlaceholderText("Search people and groups")
	box.Append(search)
	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	list := gtk.NewListBox()
	list.AddCSSClass("boxed-list")
	list.SetSelectionMode(gtk.SelectionNone)
	list.SetVAlign(gtk.AlignStart)
	scroll.SetChild(list)
	box.Append(scroll)
	empty := adw.NewStatusPage()
	empty.SetIconName("system-search-symbolic")
	empty.SetTitle("No Chats Found")
	empty.SetDescription("Try another name or group.")
	empty.SetVisible(false)
	box.Append(empty)
	selected := make(map[string]bool)
	update := func() {
		n := len(selected)
		forward.SetSensitive(n > 0)
		if n == 0 {
			forward.SetLabel("Forward")
		} else {
			forward.SetLabel(fmt.Sprintf("Forward (%d)", n))
		}
		title.SetSubtitle(fmt.Sprintf("%d message(s) · %d recipient(s)", len(messages), n))
	}
	populate := func() {
		for child := list.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
			c.releaseMessageWidget(child)
		}
		list.RemoveAll()
		query := strings.ToLower(strings.TrimSpace(search.Text()))
		visible := 0
		for _, chat := range chats {
			if !strings.Contains(strings.ToLower(formatDisplayName(chat)+" "+chat.JID), query) {
				continue
			}
			row := c.messageIdentityRow(chat.JID, formatDisplayName(chat), "")
			check := gtk.NewCheckButton()
			check.SetVAlign(gtk.AlignCenter)
			check.SetTooltipText("Forward to " + formatDisplayName(chat))
			check.SetActive(selected[chat.JID])
			c.trackWidgetSignal(check, check, check.ConnectToggled(func() {
				if check.Active() {
					selected[chat.JID] = true
				} else {
					delete(selected, chat.JID)
				}
				update()
			}))
			row.AddSuffix(check)
			row.SetActivatableWidget(check)
			row.SetActivatable(true)
			list.Append(row)
			visible++
		}
		empty.SetVisible(visible == 0)
		scroll.SetVisible(visible > 0)
	}
	c.trackWidgetSignal(win, forward, forward.ConnectClicked(func() {
		if len(selected) == 0 {
			return
		}
		var destinations []store.Chat
		for _, chat := range chats {
			if selected[chat.JID] {
				destinations = append(destinations, chat)
			}
		}
		c.cancelMessageSelection()
		win.Close()
		go func() {
			if err := forwardToDestinations(messages, destinations, c.forwardMessage); err != nil {
				glib.IdleAdd(func() { c.messageActionError(err) })
			}
		}()
	}))
	c.trackWidgetSignal(win, search, search.ConnectSearchChanged(populate))
	c.trackWidgetSignal(win, win, win.ConnectCloseRequest(func() bool { c.releaseMessageWidget(win); return false }))
	populate()
	update()
	win.Present()
	search.GrabFocus()
}

// Preserve message order per recipient, and still try other recipients after a failure.
func forwardToDestinations(messages []store.StoredMessage, destinations []store.Chat, send func(store.StoredMessage, string) error) error {
	var failures []error
	for _, destination := range destinations {
		for _, message := range messages {
			if err := send(message, destination.JID); err != nil {
				failures = append(failures, fmt.Errorf("forwarding to %s: %w", formatDisplayName(destination), err))
				break
			}
		}
	}
	return errors.Join(failures...)
}

func buildForwardMessage(msg store.StoredMessage) (*waE2E.Message, error) {
	if msg.Kind == store.MessageKindDeleted {
		return nil, fmt.Errorf("deleted messages cannot be forwarded")
	}
	var envelope waE2E.Message
	if len(msg.Raw) > 0 && proto.Unmarshal(msg.Raw, &envelope) == nil {
		candidate := &envelope
		if candidate.EphemeralMessage != nil {
			candidate = candidate.EphemeralMessage.GetMessage()
		}
		if candidate != nil && (candidate.ViewOnceMessage != nil || candidate.ViewOnceMessageV2 != nil || candidate.ViewOnceMessageV2Extension != nil) {
			return nil, fmt.Errorf("view-once messages cannot be forwarded")
		}
	}
	original := media.ParseE2EMessage(msg.Raw)
	if original == nil {
		if msg.Kind != store.MessageKindText {
			return nil, fmt.Errorf("this message has no content to forward")
		}
		original = &waE2E.Message{Conversation: proto.String(msg.Text)}
	}
	forwarded := proto.Clone(original).(*waE2E.Message)
	contextInfo := &waE2E.ContextInfo{IsForwarded: proto.Bool(true), ForwardingScore: proto.Uint32(1)}
	if old := media.ExtractContextInfoFromProto(forwarded); old != nil {
		contextInfo.ForwardingScore = proto.Uint32(old.GetForwardingScore() + 1)
		contextInfo.MentionedJID = append([]string(nil), old.GetMentionedJID()...)
	}
	switch {
	case forwarded.Conversation != nil:
		forwarded = &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: forwarded.Conversation, ContextInfo: contextInfo}}
	case forwarded.ExtendedTextMessage != nil:
		forwarded.ExtendedTextMessage.ContextInfo = contextInfo
	case forwarded.ImageMessage != nil:
		forwarded.ImageMessage.ContextInfo = contextInfo
	case forwarded.VideoMessage != nil:
		forwarded.VideoMessage.ContextInfo = contextInfo
	case forwarded.AudioMessage != nil:
		forwarded.AudioMessage.ContextInfo = contextInfo
	case forwarded.DocumentMessage != nil:
		forwarded.DocumentMessage.ContextInfo = contextInfo
	case forwarded.StickerMessage != nil:
		forwarded.StickerMessage.ContextInfo = contextInfo
	case forwarded.LocationMessage != nil:
		forwarded.LocationMessage.ContextInfo = contextInfo
	case forwarded.ContactMessage != nil:
		forwarded.ContactMessage.ContextInfo = contextInfo
	default:
		return nil, fmt.Errorf("forwarding this message type is not supported")
	}
	return forwarded, nil
}

func (c *MainController) forwardMessage(msg store.StoredMessage, destination string) error {
	waMsg, err := buildForwardMessage(msg)
	if err != nil {
		return err
	}
	jid, err := types.ParseJID(destination)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("demo_forward_%d", time.Now().UnixNano())
	if !c.store.IsDemo() {
		if c.client == nil || !c.client.IsConnected() {
			return fmt.Errorf("connect to WhatsApp to forward messages")
		}
		response, err := c.client.SendMessage(context.Background(), jid, waMsg)
		if err != nil {
			return err
		}
		id = response.ID
	}
	raw, err := proto.Marshal(waMsg)
	if err != nil {
		return err
	}
	sent := store.StoredMessage{ChatJID: destination, ID: id, FromMe: true, SenderName: "You", Timestamp: time.Now(), Kind: msg.Kind, Text: msg.Text, Raw: raw, Status: store.MessageStatusSent}
	if err := c.saveOutgoingMessage(context.Background(), sent); err != nil {
		return err
	}
	if err := c.store.UpdateChatLastMessage(context.Background(), destination, sent.Timestamp.Unix()); err != nil {
		return err
	}
	glib.IdleAdd(func() { c.appendSentMessageToUI(destination, sent) })
	return nil
}
