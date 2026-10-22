package ui

import (
	"os"
	"runtime"
	"testing"
	"time"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func TestMessageDayLabel(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, now, message, want string }{
		{"today", "2026-10-22T12:00:00+02:00", "2026-10-22T01:00:00+02:00", "Today"},
		{"yesterday", "2026-10-22T12:00:00+02:00", "2026-10-21T23:59:00+02:00", "Yesterday"},
		{"older", "2026-10-22T12:00:00+02:00", "2026-10-20T23:59:00+02:00", "20 October 2026"},
		{"year boundary", "2026-01-01T12:00:00+01:00", "2025-12-31T12:00:00+01:00", "Yesterday"},
		{"DST spring", "2026-03-30T00:30:00+02:00", "2026-03-29T00:30:00+01:00", "Yesterday"},
		{"DST autumn", "2026-10-26T23:30:00+01:00", "2026-10-25T00:30:00+02:00", "Yesterday"},
		{"local date differs from UTC", "2026-10-22T00:30:00+02:00", "2026-10-21T22:15:00Z", "Today"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			message, err := time.Parse(time.RFC3339, tc.message)
			if err != nil {
				t.Fatal(err)
			}
			if got := messageDayLabel(message, now.In(loc)); got != tc.want {
				t.Fatalf("label = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMessageGroupingIncludesDate(t *testing.T) {
	c := &MainController{}
	prev := store.StoredMessage{ChatJID: "1@s.whatsapp.net", Timestamp: time.Date(2026, 10, 21, 12, 30, 0, 0, time.Local)}
	cur := prev
	cur.Timestamp = prev.Timestamp.AddDate(0, 0, 1)
	if c.isSameAuthorAndDisplayTime(prev, cur) {
		t.Fatal("messages on different days were grouped by clock time")
	}
	cur.Timestamp = prev.Timestamp.Add(20 * time.Second)
	if !c.isSameAuthorAndDisplayTime(prev, cur) {
		t.Fatal("messages in the same minute were not grouped")
	}
}

func TestDaySeparatorsPreserveMessageRowIndices(t *testing.T) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no display available")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	gtk.Init()
	c := &MainController{}
	c.messagesList = gtk.NewListBox()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.Local)
	msgs := []store.StoredMessage{
		{ID: "yesterday-first", Timestamp: today.AddDate(0, 0, -1)},
		{ID: "yesterday-second", Timestamp: today.AddDate(0, 0, -1).Add(time.Minute)},
		{ID: "today-first", Timestamp: today},
	}
	wantTags := []string{"Yesterday", "", "Today"}
	for i, msg := range msgs {
		msg.ChatJID, msg.Kind, msg.Text = "1@s.whatsapp.net", store.MessageKindText, "Test message"
		var previous *store.StoredMessage
		if i > 0 {
			previous = &msgs[i-1]
		}
		row := c.createMessageRow(msg, previous)
		c.messagesList.Append(row)
		if coreglib.BaseObject(c.messagesList.RowAtIndex(i)).Native() != coreglib.BaseObject(row).Native() {
			t.Fatalf("day separator shifted message %d's row index", i)
		}
		var tagText string
		content := gtk.BaseWidget(row.Child())
		if tag, ok := content.FirstChild().(*gtk.Label); ok && tag.HasCSSClass("message-day-separator") {
			tagText = tag.Text()
		}
		if tagText != wantTags[i] {
			t.Fatalf("message %d tag = %q, want %q", i, tagText, wantTags[i])
		}
	}
	if c.messagesList.RowAtIndex(len(msgs)) != nil {
		t.Fatal("separators added extra message-list rows")
	}
	for child := c.messagesList.FirstChild(); child != nil; child = gtk.BaseWidget(child).NextSibling() {
		c.releaseMessageWidget(child)
	}
	c.messagesList.RemoveAll()
}
