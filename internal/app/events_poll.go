package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"log"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"postbetasoftware.com/whatsdown/internal/media"
	"postbetasoftware.com/whatsdown/internal/ui"
)

func handleIncomingPollUpdate(evt *events.Message) {
	if waClient == nil || msgStore == nil || evt.Message == nil {
		return
	}
	pollUpdate := evt.Message.GetPollUpdateMessage()
	if pollUpdate == nil || pollUpdate.GetPollCreationMessageKey() == nil {
		return
	}
	creationKey := pollUpdate.GetPollCreationMessageKey()
	pollID := creationKey.GetID()
	chatJID := evt.Info.Chat.ToNonAD().String()
	voterJID := evt.Info.Sender.ToNonAD().String()
	if evt.Info.IsFromMe && voterJID == "" && waClient.Store != nil && waClient.Store.ID != nil {
		voterJID = waClient.Store.ID.ToNonAD().String()
	}

	origMsg, err := msgStore.GetMessage(context.Background(), chatJID, pollID)
	if err != nil || origMsg == nil {
		origMsg, err = msgStore.GetMessageByID(context.Background(), pollID)
	}
	if err != nil || origMsg == nil {
		log.Printf("could not find original poll message %s in chat %s: %v", pollID, chatJID, err)
		return
	}

	ui.EnsurePollMessageSecret(context.Background(), waClient, *origMsg)
	if waMsg := media.ParseE2EMessage(origMsg.Raw); waMsg != nil {
		if sec := waMsg.GetMessageContextInfo().GetMessageSecret(); len(sec) > 0 {
			_ = waClient.Store.MsgSecrets.PutMessageSecret(context.Background(), evt.Info.Chat.ToNonAD(), evt.Info.Sender.ToNonAD(), pollID, sec)
			if creationKey.GetParticipant() != "" {
				if pJID, err := types.ParseJID(creationKey.GetParticipant()); err == nil {
					_ = waClient.Store.MsgSecrets.PutMessageSecret(context.Background(), evt.Info.Chat.ToNonAD(), pJID.ToNonAD(), pollID, sec)
				}
			}
		}
	}

	voteMsg, err := waClient.DecryptPollVote(context.Background(), evt)
	if err != nil {
		log.Printf("decrypting poll vote error: %v", err)
		return
	}

	_, options, _ := ui.ExtractPollDetails(origMsg.Raw)
	var selected []string
	for _, opt := range options {
		h := sha256.Sum256([]byte(opt))
		for _, selHash := range voteMsg.GetSelectedOptions() {
			if bytes.Equal(h[:], selHash) {
				selected = append(selected, opt)
				break
			}
		}
	}

	_ = msgStore.SetPollVotesForVoter(context.Background(), origMsg.ChatJID, pollID, voterJID, selected)
	if evt.Info.IsFromMe {
		_ = msgStore.SetPollVotesForVoter(context.Background(), origMsg.ChatJID, pollID, "me", selected)
	}
	if chatJID != origMsg.ChatJID {
		_ = msgStore.SetPollVotesForVoter(context.Background(), chatJID, pollID, voterJID, selected)
	}

	glib.IdleAdd(func() {
		if a := ui.ActiveMainWindow(); a != nil && a.CurrentChat() != nil {
			if a.CurrentChat().JID == chatJID || a.CurrentChat().JID == origMsg.ChatJID {
				a.ReloadPollBubble(pollID)
			}
		}
	})
}
