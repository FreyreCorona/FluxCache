package evict_test

import (
	"testing"

	"github.com/FreyreCorona/FluxCache/evict"
	"github.com/FreyreCorona/FluxCache/store"
)

func TestEvictionRandom(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)
	ts.SetEvictionPolicy(evict.NewRandomPolicy(), 2)

	ts.Set("a", "1")
	ts.Set("b", "2")
	_ = ts
}
