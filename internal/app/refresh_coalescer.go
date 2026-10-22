package app

import "sync/atomic"

// A replay can emit thousands of changes. Keep one pending UI refresh rather
// than retaining an idle callback and rebuilding every row for each change.
type refreshCoalescer struct{ pending atomic.Bool }

func (r *refreshCoalescer) request(schedule func(func()), refresh func()) {
	if !r.pending.CompareAndSwap(false, true) {
		return
	}
	schedule(func() {
		// Allow one trailing refresh for changes arriving during rendering.
		r.pending.Store(false)
		refresh()
	})
}
