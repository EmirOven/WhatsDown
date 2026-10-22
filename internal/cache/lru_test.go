package cache

import (
	"sync"
	"testing"
)

func TestEviction(t *testing.T) {
	c := NewLRU[string](10, 3)
	c.Put("a", "a", 4)
	c.Put("b", "b", 4)
	c.Get("a") // Reading protects a; b must be evicted first.
	c.Put("c", "c", 4)
	if _, ok := c.Get("b"); ok {
		t.Fatal("least recently used entry retained")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("recently used entry evicted")
	}
	c.Put("a", "replacement", 9)
	if c.cost != 9 || len(c.items) != 1 {
		t.Fatalf("replacement accounting: cost=%d entries=%d", c.cost, len(c.items))
	}
	c.Put("a", "oversized", 11)
	if _, ok := c.Get("a"); ok {
		t.Fatal("oversized replacement retained")
	}
	c.Put("d", "d", 1)
	c.Delete("d")
	if c.cost != 0 || len(c.items) != 0 {
		t.Fatal("delete retained an entry")
	}
}

func TestEntryLimitAndClear(t *testing.T) {
	c := NewLRU[int](100, 2)
	for _, key := range []string{"a", "b", "c"} {
		c.Put(key, 1, 1)
	}
	if len(c.items) != 2 {
		t.Fatal("entry limit exceeded")
	}
	c.Clear()
	if c.cost != 0 || len(c.items) != 0 || c.order.Len() != 0 {
		t.Fatal("clear retained entries")
	}
	c.Put("a", 2, 1)
	if v, ok := c.Get("a"); !ok || v != 2 {
		t.Fatal("cache unusable after clear")
	}
}

func TestConcurrentUse(t *testing.T) {
	c := NewLRU[int](10, 5)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			for j := 0; j < 100; j++ {
				c.Put("a", j, 1)
				c.Get("a")
				c.Delete("a")
				c.Clear()
			}
		})
	}
	wg.Wait()
}
