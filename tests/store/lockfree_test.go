package store_test

import (
	"fmt"
	"testing"

	"github.com/FreyreCorona/FluxCache/store"
)

func TestLockFreeStore(t *testing.T) {
	testStore(t, func() store.Store { return store.NewLockFreeStore(16) })
}

func BenchmarkLockFreeStoreSet(b *testing.B) {
	s := store.NewLockFreeStore(256)
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

func BenchmarkLockFreeStoreGet(b *testing.B) {
	s := store.NewLockFreeStore(256)
	for i := 0; i < 1000; i++ {
		s.Set(fmt.Sprintf("key-%d", i), "val")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(fmt.Sprintf("key-%d", i%1000))
	}
}

func BenchmarkLockFreeStoreHSet(b *testing.B) {
	s := store.NewLockFreeStore(256)
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

func BenchmarkLockFreeStoreHGetAll(b *testing.B) {
	s := store.NewLockFreeStore(256)
	s.HSet("benchhash", "a", "1")
	s.HSet("benchhash", "b", "2")
	s.HSet("benchhash", "c", "3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.HGetAll("benchhash")
	}
}
