package app

import (
	"sync"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"postbetasoftware.com/whatsdown/internal/ui"
)

var receiptRefresh struct {
	sync.Mutex
	chats     map[string]map[string]bool
	coalescer refreshCoalescer
}

func queueReceiptRefresh(chat string, ids []string) {
	receiptRefresh.Lock()
	if receiptRefresh.chats == nil {
		receiptRefresh.chats = make(map[string]map[string]bool)
	}
	if receiptRefresh.chats[chat] == nil {
		receiptRefresh.chats[chat] = make(map[string]bool)
	}
	for _, id := range ids {
		receiptRefresh.chats[chat][id] = true
	}
	receiptRefresh.Unlock()
	receiptRefresh.coalescer.request(func(run func()) { glib.TimeoutAdd(100, func() bool { run(); return false }) }, func() {
		receiptRefresh.Lock()
		chats := receiptRefresh.chats
		receiptRefresh.chats = nil
		receiptRefresh.Unlock()
		if active := ui.ActiveMainWindow(); active != nil {
			for chat, ids := range chats {
				keys := make([]string, 0, len(ids))
				for id := range ids {
					keys = append(keys, id)
				}
				active.RefreshReceiptMessages(chat, keys)
			}
		}
	})
}
