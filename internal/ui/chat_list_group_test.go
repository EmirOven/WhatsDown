package ui

import (
	"testing"
	"time"

	"postbetasoftware.com/whatsdown/internal/store"
)

func TestGroupCommunitiesPinned(t *testing.T) {
	now := time.Now()
	chats := []store.Chat{
		{
			JID:           "regular-1@s.whatsapp.net",
			Name:          "Regular Chat",
			LastMessageAt: now.Add(-5 * time.Minute),
			Pinned:        false,
		},
		{
			JID:           "comm-parent@g.us",
			Name:          "My Community",
			IsParent:      true,
			LastMessageAt: now.Add(-10 * time.Minute),
			Pinned:        false,
		},
		{
			JID:               "comm-announcements@g.us",
			Name:              "My Community",
			ParentJID:         "comm-parent@g.us",
			IsDefaultSubGroup: true,
			LastMessageAt:     now.Add(-2 * time.Minute),
			Pinned:            true, // child is pinned!
		},
	}

	expanded := make(map[string]bool)
	topLevel, children := groupCommunities(chats, "", expanded)

	if len(topLevel) != 2 {
		t.Fatalf("expected 2 topLevel chats, got %d", len(topLevel))
	}

	// Because child is pinned, the community should be marked pinned and sorted to top
	if !topLevel[0].Pinned || topLevel[0].JID != "comm-parent@g.us" {
		t.Errorf("expected community to be hoisted to top as pinned, got %v", topLevel[0])
	}

	kids := children["comm-parent@g.us"]
	if len(kids) != 1 {
		t.Fatalf("expected 1 child in community, got %d", len(kids))
	}

	// Default subgroup should format as Announcements
	disp := formatDisplayName(kids[0])
	if disp != "Announcements" {
		t.Errorf("expected default subgroup name to be 'Announcements', got %q", disp)
	}
}
