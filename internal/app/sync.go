package app

import (
	"fmt"
	"log"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// renderSync shows the current sync state on the syncing page.
func renderSync() {
	if syncProgressBar == nil {
		return
	}

	s := syncInfo
	var fraction float64 = -1
	var barText, status string

	switch {
	case s.reconnecting && s.done:
		fraction, barText = 1, "Up to date"
		if s.offlineReceived == 0 {
			status = "No new messages."
		} else if s.offlineReceived == 1 {
			status = "Received 1 new message."
		} else {
			status = fmt.Sprintf("Received %d new messages.", s.offlineReceived)
		}
	case s.reconnecting && s.offlineTotal > 0:
		received := min(s.offlineReceived, s.offlineTotal)
		fraction = float64(received) / float64(s.offlineTotal)
		barText = fmt.Sprintf("%d / %d", received, s.offlineTotal)
		status = "Fetching messages received while WhatsDown was closed…"
	case s.reconnecting:
		barText = "Connecting…"
		status = "Checking for new messages."
	case s.done:
		fraction, barText = 1, "Done"
		status = fmt.Sprintf("Synced %d chats and %d messages.", s.conversations, s.messages)
	case s.percent >= 0:
		fraction = float64(s.percent) / 100
		barText = fmt.Sprintf("%d%%", s.percent)
		status = fmt.Sprintf("%d chats and %d messages received so far…", s.conversations, s.messages)
	default:
		barText = "Waiting for your phone…"
		status = "Your phone is preparing your chat history."
	}

	syncProgressBar.SetText(barText)
	syncStatusLabel.SetText(status)
	if fraction >= 0 {
		syncProgressBar.SetFraction(fraction)
	}

	pulsing := fraction < 0
	if pulsing && syncPulse == 0 {
		syncPulse = glib.TimeoutAdd(120, func() bool {
			if syncProgressBar == nil {
				syncPulse = 0
				return false
			}
			syncProgressBar.Pulse()
			return true
		})
	} else if !pulsing && syncPulse != 0 {
		glib.SourceRemove(syncPulse)
		syncPulse = 0
	}

	if s.done && !syncTransitionScheduled {
		syncTransitionScheduled = true
		glib.TimeoutAdd(900, func() bool {
			if currentPage == pageSyncing && syncInfo.done {
				if err := showPage(pageMain); err != nil {
					log.Printf("showing main page: %v", err)
				}
			}
			return false
		})
	}
}

// handleHistorySync records a history sync chunk.
func handleHistorySync(conversations, messages, percent int) {
	syncInfo.conversations += conversations
	syncInfo.messages += messages
	if percent >= 0 {
		syncInfo.percent = percent
		if percent >= 100 {
			syncInfo.done = true
		}
	}
	renderSync()
}
