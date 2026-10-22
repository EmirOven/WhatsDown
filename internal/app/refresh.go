package app

import (
	"sync/atomic"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/ui"
)

var chatRefresh refreshCoalescer
var messageRefreshNeeded atomic.Bool

func refreshChatsUI() {
	chatRefresh.request(func(run func()) {
		glib.TimeoutAdd(100, func() bool { run(); return false })
	}, func() {
		if a := ui.ActiveMainWindow(); a != nil {
			a.LoadChats()
			a.LoadArchivedChats()
			if messageRefreshNeeded.Swap(false) {
				a.RefreshMessageRows()
			}
		}
	})
}
