package evict_test

import (
	"testing"
	"time"

	"github.com/FreyreCorona/FluxCache/evict"
	"github.com/FreyreCorona/FluxCache/store"
)

func TestEvictionTTL(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)
	ts.SetEvictionPolicy(evict.NewTTLPolicy(), 2)

	ts.SetWithTTL("a", "1", 10*time.Minute)
	ts.SetWithTTL("b", "2", 1*time.Minute)
	ts.Set("c", "3")

	_, ok := ts.Get("b")
	if ok {
		t.Fatal("expected b to be evicted (nearest TTL)")
	}
	_, ok = ts.Get("a")
	if !ok {
		t.Fatal("expected a to remain")
	}
}
