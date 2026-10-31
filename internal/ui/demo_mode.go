package ui

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

// isDemo reports whether the controller is showing the local Demo Mode store.
func (c *MainController) isDemo() bool {
	return c.store != nil && c.store.IsDemo()
}

// demoRecipients mirrors the demo group roster so Message Info has rows to show.
func (c *MainController) demoRecipients(chatJID string) []string {
	if !strings.HasSuffix(chatJID, "@g.us") {
		return []string{chatJID}
	}
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		return nil
	}
	var recipients []string
	for _, participant := range c.buildDemoGroupExtendedInfo(jid).Info.Participants {
		if target := participant.JID.ToNonAD().String(); target != store.DemoSelfJID {
			recipients = append(recipients, target)
		}
	}
	return recipients
}

// saveDemoReceipts simulates every demo recipient receiving and reading a message.
func (c *MainController) saveDemoReceipts(ctx context.Context, msg store.StoredMessage) error {
	recipients := c.demoRecipients(msg.ChatJID)
	if err := c.store.SetMessageRecipients(ctx, msg.ChatJID, msg.ID, recipients); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err := c.store.SaveMessageReceipt(ctx, msg.ChatJID, []string{msg.ID}, recipient, store.MessageStatusDelivered, msg.Timestamp); err != nil {
			return err
		}
		if err := c.store.SaveMessageReceipt(ctx, msg.ChatJID, []string{msg.ID}, recipient, store.MessageStatusRead, msg.Timestamp.Add(time.Second)); err != nil {
			return err
		}
	}
	return nil
}

// copyCachedMedia reuses an already downloaded file for a new message, such as a
// forwarded copy. Demo media cannot be downloaded again, so this keeps it visible.
func copyCachedMedia(from, to store.StoredMessage) {
	source, target := media.GetLocalMediaPath(from), media.GetLocalMediaPath(to)
	if source == target {
		return
	}
	in, err := os.Open(source)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return
	}
	_, err = io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(target)
	}
}
