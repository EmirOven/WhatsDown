package store

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func (s *MessageStore) seedDemoMessages(ctx context.Context, now time.Time) error {
	coreJID := "120363000000000001@g.us"
	aliceJID := "15550100001@s.whatsapp.net"
	marcoJID := "15550100002@s.whatsapp.net"
	elenaJID := "15550100003@s.whatsapp.net"
	liamJID := "15550100004@s.whatsapp.net"

	replyRaw, _ := proto.Marshal(&waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String("Love the new 4px color bar and window tinting! Looks super native on GNOME."),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String("demo_core_1"),
				Participant:   proto.String(aliceJID),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("Hey team! We just landed *Release 3* with chat folders, media viewers and `Message Info`, plus ~old~ new UI polish!")},
			},
		},
	})

	pollRaw, _ := proto.Marshal(&waE2E.Message{
		PollCreationMessageV3: &waE2E.PollCreationMessage{
			Name:                   proto.String("What feature should we prioritize next for Release 3?"),
			SelectableOptionsCount: proto.Uint32(1),
			Options: []*waE2E.PollCreationMessage_Option{
				{OptionName: proto.String("Voice & Video Calls")},
				{OptionName: proto.String("Animated Stickers")},
				{OptionName: proto.String("Multi-Account Support")},
			},
		},
	})

	msgs := []StoredMessage{
		{ChatJID: coreJID, ID: "demo_core_1", SenderJID: aliceJID, SenderName: "Alice Rivera", FromMe: false, Timestamp: now.Add(-25 * time.Minute), Kind: MessageKindText, Text: "Hey team! We just landed *Release 3* with chat folders, media viewers and `Message Info`, plus ~old~ new UI polish!", Starred: true},
		{ChatJID: coreJID, ID: "demo_core_2", SenderJID: marcoJID, SenderName: "Marco Rossi", FromMe: false, Timestamp: now.Add(-18 * time.Minute), Kind: MessageKindText, Text: "Love the new 4px color bar and window tinting! Looks super native on GNOME.", Raw: replyRaw},
		{ChatJID: coreJID, ID: "demo_core_3", SenderJID: DemoSelfJID, SenderName: "You", FromMe: true, Timestamp: now.Add(-10 * time.Minute), Kind: MessageKindPoll, Text: "What feature should we prioritize next for Release 3?", Raw: pollRaw, Status: MessageStatusRead},
		{ChatJID: coreJID, ID: "demo_core_4", SenderJID: elenaJID, SenderName: "Elena Rostova", FromMe: false, Timestamp: now.Add(-3 * time.Minute), Kind: MessageKindText, Text: "Demo Mode is ready for screenshots 🎉 Try selecting messages, forwarding them to several chats, or opening Message Info on your own messages!"},
		{ChatJID: aliceJID, ID: "demo_alice_1", SenderJID: DemoSelfJID, SenderName: "You", FromMe: true, Timestamp: now.Add(-45 * time.Minute), Kind: MessageKindText, Text: "Hey Alice, did you get a chance to review the new Libadwaita dialogs?", Status: MessageStatusRead},
		{ChatJID: aliceJID, ID: "demo_alice_2", SenderJID: aliceJID, SenderName: "Alice Rivera", FromMe: false, Timestamp: now.Add(-12 * time.Minute), Kind: MessageKindText, Text: "Yes! The adaptive layout and dark mode transitions look *fantastic*."},
		{ChatJID: aliceJID, ID: "demo_alice_3", SenderJID: aliceJID, SenderName: "Alice Rivera", FromMe: false, Timestamp: now.Add(-8 * time.Minute), Kind: MessageKindText, Text: "Let's grab coffee later and finalize the release notes ☕"},
		{ChatJID: marcoJID, ID: "demo_marco_1", SenderJID: marcoJID, SenderName: "Marco Rossi", FromMe: false, Timestamp: now.Add(-60 * time.Minute), Kind: MessageKindText, Text: "Are we still on for the code review at 4 PM?"},
		{ChatJID: marcoJID, ID: "demo_marco_2", SenderJID: DemoSelfJID, SenderName: "You", FromMe: true, Timestamp: now.Add(-25 * time.Minute), Kind: MessageKindText, Text: "Definitely! I'll share the benchmark numbers then.", Status: MessageStatusRead},
		{ChatJID: "120363000000000011@g.us", ID: "demo_ann_1", SenderJID: aliceJID, SenderName: "Alice Rivera", FromMe: false, Timestamp: now.Add(-45 * time.Minute), Kind: MessageKindText, Text: "📢 Welcome to the *Linux Desktop Community*! Check out the subgroups for discussions."},
		{ChatJID: "120363000000000012@g.us", ID: "demo_gtk_1", SenderJID: liamJID, SenderName: "Liam O'Connor", FromMe: false, Timestamp: now.Add(-40 * time.Minute), Kind: MessageKindText, Text: "Has anyone tried the new `Adw.Dialog` API with Blueprint? It's so clean!"},
		{ChatJID: "120363000000000002@g.us", ID: "demo_hike_1", SenderJID: liamJID, SenderName: "Liam O'Connor", FromMe: false, Timestamp: now.Add(-180 * time.Minute), Kind: MessageKindText, Text: "Trail conditions for Saturday look clear and sunny 🏔️🥾"},
		{ChatJID: elenaJID, ID: "demo_elena_1", SenderJID: elenaJID, SenderName: "Elena Rostova", FromMe: false, Timestamp: now.Add(-420 * time.Minute), Kind: MessageKindText, Text: "Sent over the SVG icon assets!"},
		{ChatJID: "120363000000000099@g.us", ID: "demo_arch_1", SenderJID: marcoJID, SenderName: "Marco Rossi", FromMe: false, Timestamp: now.Add(-2880 * time.Minute), Kind: MessageKindText, Text: "Archiving this channel now that migration is complete."},
	}
	for _, m := range msgs {
		if err := s.SaveMessage(ctx, m); err != nil {
			return err
		}
		if m.Starred {
			_ = s.ToggleMessageStarred(ctx, m.ChatJID, m.ID, true)
		}
	}
	return nil
}
