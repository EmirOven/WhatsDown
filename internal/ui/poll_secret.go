package ui

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/store"
)

// EnsurePollMessageSecret ensures the poll's 32-byte message secret is stored in MsgSecrets.
func EnsurePollMessageSecret(ctx context.Context, client *whatsmeow.Client, msg store.StoredMessage) {
	if client == nil || client.Store == nil || client.Store.MsgSecrets == nil || len(msg.Raw) == 0 {
		return
	}
	waMsg := media.ParseE2EMessage(msg.Raw)
	if waMsg == nil {
		return
	}
	msgSecret := waMsg.GetMessageContextInfo().GetMessageSecret()
	if len(msgSecret) == 0 {
		return
	}

	chatJID, _ := types.ParseJID(msg.ChatJID)
	senderJID, _ := types.ParseJID(msg.SenderJID)
	if senderJID.IsEmpty() && client.Store.ID != nil {
		senderJID = client.Store.ID.ToNonAD()
	}

	if !chatJID.IsEmpty() && !senderJID.IsEmpty() {
		_ = client.Store.MsgSecrets.PutMessageSecret(ctx, chatJID.ToNonAD(), senderJID.ToNonAD(), msg.ID, msgSecret)
	}

	if client.Store.LIDs != nil {
		storeLIDAlternates(ctx, client, chatJID, senderJID, msg.ID, msgSecret)
	}

	if client.Store.ID != nil {
		ownPN := client.Store.ID.ToNonAD()
		_ = client.Store.MsgSecrets.PutMessageSecret(ctx, ownPN, ownPN, msg.ID, msgSecret)
		if !client.Store.LID.IsEmpty() {
			ownLID := client.Store.LID.ToNonAD()
			_ = client.Store.MsgSecrets.PutMessageSecret(ctx, ownLID, ownLID, msg.ID, msgSecret)
			_ = client.Store.MsgSecrets.PutMessageSecret(ctx, ownPN, ownLID, msg.ID, msgSecret)
			_ = client.Store.MsgSecrets.PutMessageSecret(ctx, ownLID, ownPN, msg.ID, msgSecret)
		}
	}
}

func storeLIDAlternates(ctx context.Context, client *whatsmeow.Client, chatJID, senderJID types.JID, msgID string, secret []byte) {
	if altChat, err := client.Store.LIDs.GetLIDForPN(ctx, chatJID.ToNonAD()); err == nil && !altChat.IsEmpty() {
		_ = client.Store.MsgSecrets.PutMessageSecret(ctx, altChat.ToNonAD(), senderJID.ToNonAD(), msgID, secret)
	}
	if altSender, err := client.Store.LIDs.GetLIDForPN(ctx, senderJID.ToNonAD()); err == nil && !altSender.IsEmpty() {
		_ = client.Store.MsgSecrets.PutMessageSecret(ctx, chatJID.ToNonAD(), altSender.ToNonAD(), msgID, secret)
	}
	if altChat, err := client.Store.LIDs.GetPNForLID(ctx, chatJID.ToNonAD()); err == nil && !altChat.IsEmpty() {
		_ = client.Store.MsgSecrets.PutMessageSecret(ctx, altChat.ToNonAD(), senderJID.ToNonAD(), msgID, secret)
	}
	if altSender, err := client.Store.LIDs.GetPNForLID(ctx, senderJID.ToNonAD()); err == nil && !altSender.IsEmpty() {
		_ = client.Store.MsgSecrets.PutMessageSecret(ctx, chatJID.ToNonAD(), altSender.ToNonAD(), msgID, secret)
	}
}
