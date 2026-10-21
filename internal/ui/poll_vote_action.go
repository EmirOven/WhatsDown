package ui

import (
	"context"
	"log"
	"strings"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"go.mau.fi/whatsmeow/types"
	"postbetasoftware.com/whatsdown/internal/store"
)

// handlePollVote toggles a vote on a poll option and broadcasts via WhatsApp.
func (c *MainController) handlePollVote(
	msg store.StoredMessage,
	optName string,
	allowMultiple bool,
	widgets map[string]*pollOptWidget,
	totalVotesLbl *gtk.Label,
) {
	if c.store == nil {
		return
	}
	go func() {
		voterJID := "me"
		if c.client != nil && c.client.Store != nil && c.client.Store.ID != nil {
			voterJID = c.client.Store.ID.ToNonAD().String()
		}
		userVotes, err := c.store.TogglePollVote(context.Background(), msg.ChatJID, msg.ID, optName, allowMultiple, voterJID)
		if err != nil {
			log.Printf("toggling poll vote error: %v", err)
			return
		}
		if voterJID != "me" {
			var selectedList []string
			for name, sel := range userVotes {
				if sel {
					selectedList = append(selectedList, name)
				}
			}
			_ = c.store.SetPollVotesForVoter(context.Background(), msg.ChatJID, msg.ID, "me", selectedList)
		}

		c.sendPollVoteNetwork(msg, userVotes)
		results, _ := c.store.GetPollResults(context.Background(), msg.ChatJID, msg.ID)

		glib.IdleAdd(func() {
			c.updatePollOptionWidgets(widgets, totalVotesLbl, userVotes, results, allowMultiple)
		})
	}()
}

func (c *MainController) sendPollVoteNetwork(msg store.StoredMessage, userVotes map[string]bool) {
	if c.client == nil || !c.client.IsConnected() {
		return
	}
	EnsurePollMessageSecret(context.Background(), c.client, msg)
	var selectedList []string
	for name, sel := range userVotes {
		if sel {
			selectedList = append(selectedList, name)
		}
	}
	chatJID, _ := types.ParseJID(msg.ChatJID)
	senderJID, _ := types.ParseJID(msg.SenderJID)
	if (senderJID.IsEmpty() || msg.FromMe || c.isSelfJID(msg.ChatJID)) && c.client.Store != nil && c.client.Store.ID != nil {
		senderJID = c.client.Store.ID.ToNonAD()
	}
	pollInfo := &types.MessageInfo{
		ID:        types.MessageID(msg.ID),
		Chat:      chatJID.ToNonAD(),
		Sender:    senderJID.ToNonAD(),
		IsFromMe:  msg.FromMe || c.isSelfJID(msg.ChatJID),
		IsGroup:   strings.HasSuffix(msg.ChatJID, "@g.us"),
		Timestamp: msg.Timestamp,
	}
	voteMsg, err := c.client.BuildPollVote(context.Background(), pollInfo, selectedList)
	if err != nil {
		log.Printf("building poll vote error: %v", err)
	} else if voteMsg != nil {
		if resp, err := c.client.SendMessage(context.Background(), chatJID, voteMsg); err != nil {
			log.Printf("sending poll vote error: %v", err)
		} else {
			log.Printf("poll vote successfully sent: %v", resp.ID)
		}
	}
}
