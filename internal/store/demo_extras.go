package store

import (
	"context"
	"time"
)

func (s *MessageStore) seedDemoExtras(ctx context.Context, now time.Time) error {
	coreJID := "120363000000000001@g.us"
	aliceJID := "15550100001@s.whatsapp.net"
	marcoJID := "15550100002@s.whatsapp.net"
	elenaJID := "15550100003@s.whatsapp.net"

	reactions := []Reaction{
		{ChatJID: coreJID, MessageID: "demo_core_1", SenderJID: marcoJID, SenderName: "Marco Rossi", Emoji: "🚀", Timestamp: now.Add(-20 * time.Minute)},
		{ChatJID: coreJID, MessageID: "demo_core_1", SenderJID: DemoSelfJID, SenderName: "You", Emoji: "❤️", FromMe: true, Timestamp: now.Add(-19 * time.Minute)},
		{ChatJID: coreJID, MessageID: "demo_core_2", SenderJID: aliceJID, SenderName: "Alice Rivera", Emoji: "🔥", Timestamp: now.Add(-15 * time.Minute)},
		{ChatJID: coreJID, MessageID: "demo_core_4", SenderJID: aliceJID, SenderName: "Alice Rivera", Emoji: "🎉", Timestamp: now.Add(-2 * time.Minute)},
	}
	for _, r := range reactions {
		_ = s.SaveReaction(ctx, r)
	}

	_ = s.SetPollVotesForVoter(ctx, coreJID, "demo_core_3", "me", []string{"Voice & Video Calls"})
	_ = s.SetPollVotesForVoter(ctx, coreJID, "demo_core_3", DemoSelfJID, []string{"Voice & Video Calls"})
	_ = s.SetPollVotesForVoter(ctx, coreJID, "demo_core_3", aliceJID, []string{"Voice & Video Calls"})
	_ = s.SetPollVotesForVoter(ctx, coreJID, "demo_core_3", marcoJID, []string{"Animated Stickers"})
	_ = s.SetPollVotesForVoter(ctx, coreJID, "demo_core_3", elenaJID, []string{"Voice & Video Calls"})

	wallpapers := []struct{ jid, color string }{
		{coreJID, "#1c3b2b"},
		{aliceJID, "#2b1e3f"},
		{"120363000000000012@g.us", "#1e2d3b"},
	}
	for _, w := range wallpapers {
		_ = s.SetChatWallpaper(ctx, w.jid, "color", w.color)
	}
	_ = s.SetAppSetting(ctx, "tinting_enabled", "true")

	calls := []CallRecord{
		{ID: "demo_call_1", ChatJID: aliceJID, CallerJID: aliceJID, CallerName: "Alice Rivera", Timestamp: now.Add(-50 * time.Minute), DurationSecs: 845, IsVideo: true, Outcome: CallOutcomeConnected, FromMe: false},
		{ID: "demo_call_2", ChatJID: marcoJID, CallerJID: marcoJID, CallerName: "Marco Rossi", Timestamp: now.Add(-3 * time.Hour), DurationSecs: 0, IsVideo: false, Outcome: CallOutcomeMissed, FromMe: false},
		{ID: "demo_call_3", ChatJID: coreJID, CallerJID: DemoSelfJID, CallerName: "WhatsDown Core Team", Timestamp: now.Add(-26 * time.Hour), DurationSecs: 1920, IsVideo: true, IsGroup: true, Outcome: CallOutcomeConnected, FromMe: true},
	}
	for _, c := range calls {
		_ = s.SaveCall(ctx, c)
	}
	return nil
}
