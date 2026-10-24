package ui

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/store"
)

// Save outgoing messages and capture the group members independently of receipts.
func (c *MainController) saveOutgoingMessage(ctx context.Context, msg store.StoredMessage) error {
	if err := c.store.SaveMessage(ctx, msg); err != nil {
		return err
	}
	if !strings.HasSuffix(msg.ChatJID, "@g.us") {
		return c.store.SetMessageRecipients(ctx, msg.ChatJID, msg.ID, []string{msg.ChatJID})
	}
	if c.client == nil || c.store.IsDemo() {
		return nil
	}
	jid, err := types.ParseJID(msg.ChatJID)
	if err != nil {
		return err
	}
	group, err := c.client.GetGroupInfo(ctx, jid)
	if err != nil {
		log.Printf("loading message recipients: %v", err)
		return nil
	}
	var recipients []string
	for _, participant := range group.Participants {
		target := participant.JID.ToNonAD().String()
		if participant.PhoneNumber.User != "" {
			target = participant.PhoneNumber.ToNonAD().String()
		}
		if c.isSelfJID(target) || c.isSelfJID(participant.JID.ToNonAD().String()) {
			continue
		}
		recipients = append(recipients, target)
	}
	return c.store.SetMessageRecipients(ctx, msg.ChatJID, msg.ID, recipients)
}

type messageInfoView struct {
	window             *gtk.Window
	refresh            func()
	received           func()
	chatJID, messageID string
}

func (c *MainController) showMessageInfo(msg store.StoredMessage) {
	if c.store == nil || !msg.FromMe {
		return
	}
	view := c.buildMessageInfo(msg)
	view.window.Present()
}

func (c *MainController) buildMessageInfo(msg store.StoredMessage) *messageInfoView {
	win, box, header := c.newMessageDialog("Message Info", "")
	header.SetTitleWidget(adw.NewWindowTitle("Message Info", msg.Timestamp.Local().Format("2 January 2006, 15:04")))
	scroll := gtk.NewScrolledWindow()
	scroll.SetVExpand(true)
	scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	content := gtk.NewBox(gtk.OrientationVertical, 24)
	scroll.SetChild(content)
	box.Append(scroll)
	summaryList := gtk.NewListBox()
	summaryList.SetSelectionMode(gtk.SelectionNone)
	summaryList.AddCSSClass("boxed-list")
	summary := adw.NewActionRow()
	summary.SetUseMarkup(false)
	preview := truncatePreviewLines(msg.Text, 3)
	if preview == "" {
		preview = msg.Kind
	}
	summary.SetTitle(preview)
	summary.SetTitleLines(3)
	summary.SetSubtitle("Sent at " + msg.Timestamp.Local().Format("15:04"))
	summary.AddPrefix(gtk.NewImageFromIconName("mail-send-symbolic"))
	summaryList.Append(summary)
	content.Append(summaryList)
	notice := gtk.NewLabel("WhatsApp has not supplied a complete recipient list for this message. Only recipients reported by WhatsApp are shown.")
	notice.SetWrap(true)
	notice.SetXAlign(0)
	notice.AddCSSClass("dim-label")
	notice.AddCSSClass("caption")
	content.Append(notice)
	syncRow := gtk.NewBox(gtk.OrientationHorizontal, 12)
	syncStatus := gtk.NewLabel("Live WhatsApp receipts update automatically.")
	syncStatus.SetWrap(true)
	syncStatus.SetXAlign(0)
	syncStatus.SetHExpand(true)
	syncStatus.AddCSSClass("dim-label")
	syncStatus.AddCSSClass("caption")
	syncButton := gtk.NewButtonWithLabel("Sync from phone")
	syncButton.SetVAlign(gtk.AlignCenter)
	syncRow.Append(syncStatus)
	syncRow.Append(syncButton)
	content.Append(syncRow)
	sections := gtk.NewBox(gtk.OrientationVertical, 24)
	content.Append(sections)
	errorLabel := gtk.NewLabel("")
	errorLabel.SetWrap(true)
	errorLabel.AddCSSClass("error")
	errorLabel.SetVisible(false)
	content.Append(errorLabel)
	var previous []store.MessageReceipt
	var previousKnown bool
	loaded := false
	refresh := func() {
		receipts, known, err := c.store.GetMessageReceipts(context.Background(), msg.ChatJID, msg.ID)
		if err != nil {
			errorLabel.SetText("Could not update receipts. Retrying automatically…")
			errorLabel.SetVisible(true)
			return
		}
		errorLabel.SetVisible(false)
		if loaded && known == previousKnown && reflect.DeepEqual(receipts, previous) {
			return
		}
		previous, previousKnown, loaded = receipts, known, true
		notice.SetVisible(!known)
		for child := sections.FirstChild(); child != nil; child = sections.FirstChild() {
			sections.Remove(child)
		}
		for _, section := range []struct {
			title  string
			status int
		}{
			{"Read by", store.MessageStatusRead},
			{"Delivered to", store.MessageStatusDelivered},
			{"Pending Delivery", store.MessageStatusSent},
		} {
			var matches []store.MessageReceipt
			for _, receipt := range receipts {
				if receipt.Status == section.status {
					matches = append(matches, receipt)
				}
			}
			group := adw.NewPreferencesGroup()
			group.SetTitle(fmt.Sprintf("%s · %d", section.title, len(matches)))
			list := gtk.NewListBox()
			list.SetSelectionMode(gtk.SelectionNone)
			list.AddCSSClass("boxed-list")
			for _, receipt := range matches {
				name := c.resolveContactName(receipt.RecipientJID)
				if name == "" {
					name = cleanPhoneNumber(receipt.RecipientJID)
				}
				var times []string
				if !receipt.ReadAt.IsZero() {
					times = append(times, "Read "+receipt.ReadAt.Local().Format("2 Jan, 15:04:05"))
				}
				if !receipt.DeliveredAt.IsZero() {
					times = append(times, "Delivered "+receipt.DeliveredAt.Local().Format("2 Jan, 15:04:05"))
				}
				if len(times) == 0 {
					switch receipt.Status {
					case store.MessageStatusRead:
						times = append(times, "Read")
					case store.MessageStatusDelivered:
						times = append(times, "Delivered")
					default:
						times = append(times, "No delivery receipt received")
					}
				}
				row := c.messageIdentityRow(receipt.RecipientJID, name, strings.Join(times, "\n"))
				messageButton := gtk.NewButtonWithLabel("Message")
				messageButton.AddCSSClass("flat")
				messageButton.SetVAlign(gtk.AlignCenter)
				target := receipt.RecipientJID
				messageButton.ConnectClicked(func() {
					chat, err := c.store.GetChatOrContact(context.Background(), target)
					if err != nil {
						c.messageActionError(err)
						return
					}
					if err := c.store.UpsertChat(context.Background(), chat.JID, chat.Name, 0); err != nil {
						c.messageActionError(err)
						return
					}
					win.Close()
					c.loadChats()
					c.selectChat(chat)
				})
				row.AddSuffix(messageButton)
				list.Append(row)
			}
			if len(matches) == 0 {
				row := adw.NewActionRow()
				row.SetTitle("None yet")
				row.AddCSSClass("dim-label")
				list.Append(row)
			}
			group.Add(list)
			sections.Append(group)
		}
	}

	view := &messageInfoView{window: win, refresh: refresh, chatJID: msg.ChatJID, messageID: msg.ID}
	if c.messageInfoViews == nil {
		c.messageInfoViews = make(map[*messageInfoView]struct{})
	}
	c.messageInfoViews[view] = struct{}{}
	alive, pending := true, false
	var cancelRequest context.CancelFunc
	var responseTimer glib.SourceHandle
	finish := func(text string) {
		pending = false
		if cancelRequest != nil {
			cancelRequest()
			cancelRequest = nil
		}
		if responseTimer != 0 {
			glib.SourceRemove(responseTimer)
			responseTimer = 0
		}
		syncButton.SetSensitive(true)
		syncStatus.SetText(text)
	}
	view.received = func() {
		if pending {
			finish("Updated message metadata received from your phone.")
		}
		refresh()
	}
	syncFromPhone := func() {
		if pending || !alive {
			return
		}
		if c.client == nil || !c.client.IsConnected() || c.store.IsDemo() {
			syncStatus.SetText("Connect to WhatsApp to sync message metadata from your phone.")
			return
		}
		chat, err := types.ParseJID(msg.ChatJID)
		if err != nil {
			syncStatus.SetText("Could not identify this chat.")
			return
		}
		pending = true
		syncButton.SetSensitive(false)
		syncStatus.SetText("Requesting updated message metadata from your phone…")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		cancelRequest = cancel
		responseTimer = glib.TimeoutAdd(20000, func() bool {
			responseTimer = 0
			finish("Your phone has not returned updated metadata. Received receipts are still shown.")
			return false
		})
		go func() {
			// Use whatsmeow's primary-device message-copy protocol. The response is a
			// normal Message event with SourceWebMsg/UserReceipt, imported by the store.
			_, err := c.client.SendPeerMessage(ctx, c.client.BuildUnavailableMessageRequest(chat, types.EmptyJID, msg.ID))
			glib.IdleAdd(func() {
				if !alive || !pending {
					return
				}
				if err != nil {
					finish("Could not request updated metadata from your phone. Try again.")
					return
				}
				syncStatus.SetText("Waiting for updated message metadata from your phone…")
			})
		}()
	}
	syncButton.ConnectClicked(syncFromPhone)
	refresh()
	// Cover history sync as well as live receipts without rebuilding unchanged rows.
	timer := glib.TimeoutAdd(1000, func() bool { refresh(); return true })
	stop := func() {
		alive = false
		delete(c.messageInfoViews, view)
		if cancelRequest != nil {
			cancelRequest()
			cancelRequest = nil
		}
		if responseTimer != 0 {
			glib.SourceRemove(responseTimer)
			responseTimer = 0
		}
		if timer != 0 {
			glib.SourceRemove(timer)
			timer = 0
		}
	}
	win.ConnectCloseRequest(func() bool { stop(); return false })
	win.ConnectDestroy(stop)
	if c.client != nil && c.client.IsConnected() && !c.store.IsDemo() {
		syncFromPhone()
	}
	return view
}
