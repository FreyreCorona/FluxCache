package store_test

import (
	"fmt"
	"testing"

	"github.com/FreyreCorona/FluxCache/store"
)

func TestMapStore(t *testing.T) {
	testStore(t, func() store.Store { return store.NewMapStore() })
}

func BenchmarkMapStoreSet(b *testing.B) {
	s := store.NewMapStore()
	keys := make([]string, b.N)
	vals := make([]string, b.N)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
		vals[i] = fmt.Sprintf("val-%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Set(keys[i], vals[i])
	}
}

func BenchmarkMapStoreGet(b *testing.B) {
	s := store.NewMapStore()
	keys := make([]string, b.N)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
		s.Set(keys[i], "val")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(keys[i])
	}
}

func BenchmarkMapStoreHSet(b *testing.B) {
	s := store.NewMapStore()
	hashes := make([]string, b.N)
	fields := make([]string, b.N)
	vals := make([]string, b.N)
	for i := range hashes {
		hashes[i] = fmt.Sprintf("hash-%d", i%100)
		fields[i] = fmt.Sprintf("field-%d", i)
		vals[i] = fmt.Sprintf("val-%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.HSet(hashes[i], fields[i], vals[i])
	}
}

func BenchmarkMapStoreHGetAll(b *testing.B) {
	s := store.NewMapStore()
	s.HSet("benchhash", "a", "1")
	s.HSet("benchhash", "b", "2")
	s.HSet("benchhash", "c", "3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.HGetAll("benchhash")
	}
}
