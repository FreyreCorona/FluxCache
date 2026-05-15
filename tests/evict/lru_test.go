package evict_test

import (
	"testing"

	"github.com/FreyreCorona/FluxCache/evict"
	"github.com/FreyreCorona/FluxCache/store"
)

func TestEvictionLRU(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)
	ts.SetEvictionPolicy(evict.NewLRUPolicy(), 2)

	ts.Set("a", "1")
	ts.Set("b", "2")
	ts.Get("a")
	ts.Get("a")
	ts.Set("c", "3")

	_, ok := ts.Get("b")
	if ok {
		t.Fatal("expected b to be evicted (LRU)")
	}
	_, ok = ts.Get("a")
	if !ok {
		t.Fatal("expected a to remain (accessed twice)")
	}
}

func TestEvictionMaxKeysZero(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)
	ts.SetEvictionPolicy(evict.NewLRUPolicy(), 0)

	for i := 0; i < 100; i++ {
		ts.Set(string(rune('a'+i%26)), "v")
	}
	_ = ts
}
