package evict_test

import (
	"testing"

	"github.com/FreyreCorona/FluxCache/evict"
	"github.com/FreyreCorona/FluxCache/store"
)

func TestEvictionLFU(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)
	ts.SetEvictionPolicy(evict.NewLFUPolicy(), 2)

	ts.Set("a", "1")
	ts.Set("b", "2")
	ts.Get("a")
	ts.Get("a")
	ts.Get("a")
	ts.Set("c", "3")

	_, okB := ts.Get("b")
	_, okC := ts.Get("c")
	if okB && okC {
		t.Fatal("expected one key to be evicted (both freq=1)")
	}
}
