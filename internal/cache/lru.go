// Package cache provides bounded in-memory caches for native image resources.
package cache

import (
	"container/list"
	"sync"
)

// LRU bounds both retained cost and entry count. Eviction drops Go references;
// callers and GTK widgets keep ownership of any values they are still using.
type LRU[T any] struct {
	mu            sync.Mutex
	maxCost, cost int64
	maxEntries    int
	items         map[string]*list.Element
	order         list.List
}

type entry[T any] struct {
	key   string
	value T
	cost  int64
}

func NewLRU[T any](maxCost int64, maxEntries int) *LRU[T] {
	return &LRU[T]{maxCost: maxCost, maxEntries: maxEntries, items: make(map[string]*list.Element)}
}

func (c *LRU[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.items[key]; e != nil {
		c.order.MoveToFront(e)
		return e.Value.(entry[T]).value, true
	}
	var zero T
	return zero, false
}

func (c *LRU[T]) Put(key string, value T, cost int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.items[key]; e != nil {
		c.remove(e)
	}
	if cost <= 0 || cost > c.maxCost || c.maxEntries <= 0 {
		return
	}
	for c.cost > c.maxCost-cost || len(c.items) >= c.maxEntries {
		c.remove(c.order.Back())
	}
	c.items[key] = c.order.PushFront(entry[T]{key, value, cost})
	c.cost += cost
}

func (c *LRU[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.items[key]; e != nil {
		c.remove(e)
	}
}

func (c *LRU[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.items)
	c.order.Init()
	c.cost = 0
}

func (c *LRU[T]) remove(e *list.Element) {
	v := e.Value.(entry[T])
	delete(c.items, v.key)
	c.cost -= v.cost
	c.order.Remove(e)
}
