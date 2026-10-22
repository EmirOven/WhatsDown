package ui

import (
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"postbetasoftware.com/whatsdown/internal/store"
)

func sameMessageDay(a, b time.Time) bool {
	return a.Local().Format("2006-01-02") == b.Local().Format("2006-01-02")
}

func messageDayLabel(timestamp, now time.Time) string {
	day := timestamp.In(now.Location())
	switch day.Format("2006-01-02") {
	case now.Format("2006-01-02"):
		return "Today"
	case now.AddDate(0, 0, -1).Format("2006-01-02"):
		return "Yesterday"
	default:
		return day.Format("2 January 2006")
	}
}

// Keep the day tag inside the first message's row so search and reply jumps
// retain their one-row-per-message indexing.
func (c *MainController) createMessageRow(msg store.StoredMessage, previous *store.StoredMessage) *gtk.ListBoxRow {
	sameMinute := previous != nil && c.isSameAuthorAndDisplayTime(*previous, msg)
	row := c.createMessageBubble(msg, sameMinute)
	if previous != nil && sameMessageDay(previous.Timestamp, msg.Timestamp) {
		return row
	}

	tag := gtk.NewLabel(messageDayLabel(msg.Timestamp, time.Now()))
	tag.AddCSSClass("message-day-separator")
	tag.SetHAlign(gtk.AlignCenter)
	tag.SetMarginTop(8)
	tag.SetMarginBottom(12)

	content := row.Child()
	row.SetChild(nil)
	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetHExpand(true)
	box.Append(tag)
	box.Append(content)
	row.SetChild(box)
	return row
}
