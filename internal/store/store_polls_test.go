package store_test

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestPollVotingStore(t *testing.T) {
	s, ctx, db := setupTestStore(t)
	defer db.Close()

	chatJID := types.NewJID("5551234", types.DefaultUserServer)
	pollMsg := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chatJID, Sender: chatJID, IsFromMe: false},
			ID:            "poll-msg-1",
			PushName:      "Charlie",
			Timestamp:     time.Now(),
		},
		Message: &waE2E.Message{
			PollCreationMessage: &waE2E.PollCreationMessage{
				Name: proto.String("Lunch?"),
				Options: []*waE2E.PollCreationMessage_Option{
					{OptionName: proto.String("Pizza")},
					{OptionName: proto.String("Burger")},
				},
			},
		},
	}
	if err := s.SaveLiveMessage(ctx, pollMsg); err != nil {
		t.Fatalf("save: %v", err)
	}

	votes, err := s.TogglePollVote(ctx, chatJID.String(), "poll-msg-1", "Pizza", false, "voter1")
	if err != nil || !votes["Pizza"] {
		t.Fatalf("expected Pizza voted, got %+v (err: %v)", votes, err)
	}

	results, err := s.GetPollResults(ctx, chatJID.String(), "poll-msg-1")
	if err != nil || results.TotalVoters != 1 || results.VoteCounts["Pizza"] != 1 {
		t.Fatalf("unexpected results: %+v (err: %v)", results, err)
	}

	votes, err = s.TogglePollVote(ctx, chatJID.String(), "poll-msg-1", "Pizza", false, "voter1")
	if err != nil || votes["Pizza"] {
		t.Fatalf("expected Pizza unvoted, got %+v", votes)
	}
}
