package app

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRefreshCoalescesSyncEventBurst(t *testing.T) {
	var r refreshCoalescer
	var mu sync.Mutex
	var queued []func()
	var refreshed atomic.Int32
	schedule := func(run func()) { mu.Lock(); queued = append(queued, run); mu.Unlock() }
	refresh := func() { refreshed.Add(1) }
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Go(func() {
			for j := 0; j < 10000; j++ {
				r.request(schedule, refresh)
			}
		})
	}
	wg.Wait()
	if len(queued) != 1 {
		t.Fatalf("160,000 events queued %d refreshes, want 1", len(queued))
	}
	queued[0]()
	if refreshed.Load() != 1 {
		t.Fatal("refresh not executed")
	}
	r.request(schedule, refresh)
	if len(queued) != 2 {
		t.Fatal("later changes failed to schedule a refresh")
	}
	queued[1]()
}

func TestRefreshSchedulesTrailingChange(t *testing.T) {
	var r refreshCoalescer
	var queued []func()
	schedule := func(run func()) { queued = append(queued, run) }
	r.request(schedule, func() {
		for i := 0; i < 10000; i++ {
			r.request(schedule, func() {})
		}
	})
	queued[0]()
	if len(queued) != 2 {
		t.Fatalf("changes during rendering queued %d callbacks, want 2 including original", len(queued))
	}
	queued[1]()
}
