package store

import (
	"context"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

// Demo media assets, written to the media cache by the app package because the
// store cannot depend on the media package.
const (
	DemoAssetSummitPhoto  = "summit.jpg"
	DemoAssetTrailVideo   = "trail.mp4"
	DemoAssetVoiceNote    = "voice.ogg"
	DemoAssetReleaseNotes = "release-3-notes.pdf"
)

const (
	demoCoreJID   = "120363000000000001@g.us"
	demoHikeJID   = "120363000000000002@g.us"
	demoAliceJID  = "15550100001@s.whatsapp.net"
	demoMarcoJID  = "15550100002@s.whatsapp.net"
	demoElenaJID  = "15550100003@s.whatsapp.net"
	demoLiamJID   = "15550100004@s.whatsapp.net"
	demoSophiaJID = "15550100005@s.whatsapp.net"
	demoNoahJID   = "15550100006@s.whatsapp.net"
)

// DemoMediaAsset links a seeded demo message to the embedded file it displays.
type DemoMediaAsset struct {
	Message StoredMessage
	Asset   string
}

func demoRaw(m *waE2E.Message) []byte {
	raw, _ := proto.Marshal(m)
	return raw
}

// demoVoiceWaveform imitates the 64-bar waveform WhatsApp stores for voice notes.
func demoVoiceWaveform() []byte {
	bars := []byte{8, 20, 46, 70, 62, 38, 54, 82, 90, 64, 30, 12, 26, 58, 76, 88}
	wave := make([]byte, 64)
	for i := range wave {
		wave[i] = bars[(i*7)%len(bars)]
	}
	return wave
}

// DemoMediaAssets returns the seeded Release 3 media messages. Cache paths depend
// only on the chat, ID and payload type, so timestamps are irrelevant here.
func DemoMediaAssets(now time.Time) []DemoMediaAsset {
	notes := func(caption string, forwarded bool) []byte {
		doc := &waE2E.DocumentMessage{
			FileName:   proto.String("WhatsDown Release 3 notes.pdf"),
			Title:      proto.String("WhatsDown Release 3 notes"),
			Caption:    proto.String(caption),
			Mimetype:   proto.String("application/pdf"),
			FileLength: proto.Uint64(4332),
			PageCount:  proto.Uint32(3),
		}
		if forwarded {
			doc.ContextInfo = &waE2E.ContextInfo{IsForwarded: proto.Bool(true), ForwardingScore: proto.Uint32(1)}
		}
		return demoRaw(&waE2E.Message{DocumentMessage: doc})
	}
	return []DemoMediaAsset{
		{Asset: DemoAssetSummitPhoto, Message: StoredMessage{
			ChatJID: demoAliceJID, ID: "demo_r3_alice_photo", SenderJID: demoAliceJID, SenderName: "Alice Rivera",
			Timestamp: now.Add(-27 * time.Hour), Kind: MessageKindImage, Text: "Sunset from the summit 🌄 Open it to zoom and rotate!",
			Raw: demoRaw(&waE2E.Message{ImageMessage: &waE2E.ImageMessage{
				Caption: proto.String("Sunset from the summit 🌄 Open it to zoom and rotate!"), Mimetype: proto.String("image/jpeg"),
				Width: proto.Uint32(1280), Height: proto.Uint32(853), FileLength: proto.Uint64(20946),
			}}),
		}},
		{Asset: DemoAssetVoiceNote, Message: StoredMessage{
			ChatJID: demoAliceJID, ID: "demo_r3_alice_voice", SenderJID: demoAliceJID, SenderName: "Alice Rivera",
			Timestamp: now.Add(-10 * time.Minute), Kind: MessageKindVoice,
			Raw: demoRaw(&waE2E.Message{AudioMessage: &waE2E.AudioMessage{
				Mimetype: proto.String("audio/ogg; codecs=opus"), Seconds: proto.Uint32(14), PTT: proto.Bool(true),
				Waveform: demoVoiceWaveform(), FileLength: proto.Uint64(36130),
			}}),
		}},
		{Asset: DemoAssetTrailVideo, Message: StoredMessage{
			ChatJID: demoHikeJID, ID: "demo_r3_hike_video", SenderJID: demoNoahJID, SenderName: "Noah Williams",
			Timestamp: now.Add(-200 * time.Minute), Kind: MessageKindVideo, Text: "Trail preview for Saturday 🎥",
			Raw: demoRaw(&waE2E.Message{VideoMessage: &waE2E.VideoMessage{
				Caption: proto.String("Trail preview for Saturday 🎥"), Mimetype: proto.String("video/mp4"),
				Width: proto.Uint32(854), Height: proto.Uint32(480), Seconds: proto.Uint32(6), FileLength: proto.Uint64(60309),
			}}),
		}},
		{Asset: DemoAssetReleaseNotes, Message: StoredMessage{
			ChatJID: demoCoreJID, ID: "demo_r3_core_notes", SenderJID: DemoSelfJID, SenderName: "You", FromMe: true,
			Timestamp: now.Add(-30 * time.Minute), Kind: MessageKindDocument, Text: "Release notes draft, three pages 📄",
			Raw: notes("Release notes draft, three pages 📄", false), Status: MessageStatusSent,
		}},
		{Asset: DemoAssetReleaseNotes, Message: StoredMessage{
			ChatJID: demoNoahJID, ID: "demo_r3_noah_notes", SenderJID: DemoSelfJID, SenderName: "You", FromMe: true,
			Timestamp: now.Add(-95 * time.Minute), Kind: MessageKindDocument, Text: "WhatsDown Release 3 notes.pdf",
			Raw: notes("", true), Status: MessageStatusSent,
		}},
	}
}

// seedDemoRelease3 showcases Release 3: folders, media viewers, voice playback,
// links, day separators, Message Info receipts and group sender previews.
func (s *MessageStore) seedDemoRelease3(ctx context.Context, now time.Time) error {
	chats := []struct {
		jid, name       string
		minsAgo, unread int
	}{
		{demoSophiaJID, "Sophia Chen", 15, 1},
		{demoNoahJID, "Noah Williams", 95, 0},
	}
	for _, ch := range chats {
		if _, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO whatsdown_chats (jid, name, last_message_at, unread_count) VALUES (?, ?, ?, ?)`,
			ch.jid, ch.name, now.Add(-time.Duration(ch.minsAgo)*time.Minute).Unix(), ch.unread); err != nil {
			return err
		}
	}

	msgs := []StoredMessage{
		// Day separators: three days ago, Yesterday and Today.
		{ChatJID: demoCoreJID, ID: "demo_r3_core_kickoff", SenderJID: demoElenaJID, SenderName: "Elena Rostova", Timestamp: now.Add(-74 * time.Hour), Kind: MessageKindText, Text: "Release 3 kick-off is on Monday 🚀 Day separators make it easy to see when we planned it."},
		{ChatJID: demoCoreJID, ID: "demo_r3_core_link", SenderJID: demoMarcoJID, SenderName: "Marco Rossi", Timestamp: now.Add(-26 * time.Hour), Kind: MessageKindText, Text: "Release 3 checklist is up, links are clickable now: https://github.com/EmirOven/WhatsDown/blob/main/CHANGELOG.md"},
		{ChatJID: demoCoreJID, ID: "demo_r3_core_thanks", SenderJID: DemoSelfJID, SenderName: "You", FromMe: true, Timestamp: now.Add(-25*time.Hour - 30*time.Minute), Kind: MessageKindText, Text: "Thanks! I'll go through the new media viewers tonight.", Status: MessageStatusSent},
		{ChatJID: demoAliceJID, ID: "demo_r3_alice_wow", SenderJID: DemoSelfJID, SenderName: "You", FromMe: true, Timestamp: now.Add(-26*time.Hour - 50*time.Minute), Kind: MessageKindText, Text: "Wow, that's gorgeous! 😍", Status: MessageStatusSent},
		{ChatJID: demoMarcoJID, ID: "demo_r3_marco_link", SenderJID: demoMarcoJID, SenderName: "Marco Rossi", Timestamp: now.Add(-58 * time.Minute), Kind: MessageKindText, Text: "Benchmark runs are here if you want a head start: https://github.com/EmirOven/WhatsDown/actions"},
		{ChatJID: demoHikeJID, ID: "demo_r3_hike_sophia", SenderJID: demoSophiaJID, SenderName: "Sophia Chen", Timestamp: now.Add(-190 * time.Minute), Kind: MessageKindText, Text: "Count me in! 🙋‍♀️"},
		{ChatJID: demoSophiaJID, ID: "demo_r3_sophia_1", SenderJID: demoSophiaJID, SenderName: "Sophia Chen", Timestamp: now.Add(-15 * time.Minute), Kind: MessageKindText, Text: "Did you see WhatsDown has chat folders now? 📁 Drag chats between them, or right-click a chat and choose Add to Folder."},
		{ChatJID: demoNoahJID, ID: "demo_r3_noah_1", SenderJID: demoNoahJID, SenderName: "Noah Williams", Timestamp: now.Add(-100 * time.Minute), Kind: MessageKindText, Text: "Could you forward me the Release 3 notes?"},
	}
	for _, asset := range DemoMediaAssets(now) {
		msgs = append(msgs, asset.Message)
	}
	for _, m := range msgs {
		if err := s.SaveMessage(ctx, m); err != nil {
			return err
		}
	}

	// Per-recipient receipts for Message Info, including partially delivered messages.
	type receipt struct {
		jid           string
		status, after int // minutes after the message was sent
	}
	receipts := []struct {
		chat, id string
		sent     time.Time
		list     []receipt
	}{
		{demoCoreJID, "demo_core_3", now.Add(-10 * time.Minute), []receipt{{demoAliceJID, MessageStatusRead, 1}, {demoMarcoJID, MessageStatusRead, 2}, {demoElenaJID, MessageStatusRead, 4}, {demoLiamJID, MessageStatusRead, 6}}},
		{demoCoreJID, "demo_r3_core_notes", now.Add(-30 * time.Minute), []receipt{{demoAliceJID, MessageStatusRead, 2}, {demoMarcoJID, MessageStatusRead, 9}, {demoElenaJID, MessageStatusDelivered, 1}, {demoLiamJID, MessageStatusDelivered, 3}}},
		{demoCoreJID, "demo_r3_core_thanks", now.Add(-25*time.Hour - 30*time.Minute), []receipt{{demoAliceJID, MessageStatusRead, 5}, {demoMarcoJID, MessageStatusRead, 12}, {demoElenaJID, MessageStatusRead, 40}, {demoLiamJID, MessageStatusRead, 95}}},
		{demoAliceJID, "demo_alice_1", now.Add(-45 * time.Minute), []receipt{{demoAliceJID, MessageStatusRead, 20}}},
		{demoAliceJID, "demo_r3_alice_wow", now.Add(-26*time.Hour - 50*time.Minute), []receipt{{demoAliceJID, MessageStatusRead, 2}}},
		{demoMarcoJID, "demo_marco_2", now.Add(-25 * time.Minute), []receipt{{demoMarcoJID, MessageStatusRead, 3}}},
		{demoNoahJID, "demo_r3_noah_notes", now.Add(-95 * time.Minute), []receipt{{demoNoahJID, MessageStatusDelivered, 1}}},
	}
	for _, r := range receipts {
		recipients := make([]string, 0, len(r.list))
		for _, item := range r.list {
			recipients = append(recipients, item.jid)
		}
		if err := s.SetMessageRecipients(ctx, r.chat, r.id, recipients); err != nil {
			return err
		}
		for _, item := range r.list {
			at := r.sent.Add(time.Duration(item.after) * time.Minute)
			if item.status == MessageStatusRead {
				// A read receipt implies earlier delivery; record both timestamps.
				if err := s.SaveMessageReceipt(ctx, r.chat, []string{r.id}, item.jid, MessageStatusDelivered, r.sent.Add(30*time.Second)); err != nil {
					return err
				}
			}
			if err := s.SaveMessageReceipt(ctx, r.chat, []string{r.id}, item.jid, item.status, at); err != nil {
				return err
			}
		}
	}

	folders := []struct {
		name, color string
		jids        []string
	}{
		{"Work", "#3584e4", []string{demoMarcoJID, demoElenaJID, demoNoahJID}},
		{"Friends", "#e66100", []string{demoSophiaJID, demoHikeJID}},
	}
	for _, f := range folders {
		if _, err := s.CreateChatFolder(ctx, f.name, f.color, f.jids); err != nil {
			return err
		}
	}
	return nil
}
