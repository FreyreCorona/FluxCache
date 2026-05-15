package store_test

import (
	"fmt"
	"testing"

	"github.com/FreyreCorona/FluxCache/store"
)

func TestARTStore(t *testing.T) {
	testStore(t, func() store.Store { return store.NewARTStore() })
}

func TestARTStoreOrdered(t *testing.T) {
	testOrderedStore(t, func() store.OrderedStore { return store.NewARTStore() })
}

func BenchmarkARTStoreSet(b *testing.B) {
	s := store.NewARTStore()
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

func BenchmarkARTStoreGet(b *testing.B) {
	s := store.NewARTStore()
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

func BenchmarkARTStoreHSet(b *testing.B) {
	s := store.NewARTStore()
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

func BenchmarkARTStoreHGetAll(b *testing.B) {
	s := store.NewARTStore()
	s.HSet("benchhash", "a", "1")
	s.HSet("benchhash", "b", "2")
	s.HSet("benchhash", "c", "3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.HGetAll("benchhash")
	}
}
