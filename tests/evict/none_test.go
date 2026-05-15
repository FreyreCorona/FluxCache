package evict_test

import (
	"testing"

	"github.com/FreyreCorona/FluxCache/evict"
	"github.com/FreyreCorona/FluxCache/store"
)

func TestEvictionNoEviction(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)
	ts.SetEvictionPolicy(evict.NewNoEviction(), 2)

	ts.Set("a", "1")
	ts.Set("b", "2")
	ts.Set("c", "3")

	_, ok := ts.Get("c")
	if !ok {
		t.Fatal("expected c to exist (NoEviction)")
	}
}
