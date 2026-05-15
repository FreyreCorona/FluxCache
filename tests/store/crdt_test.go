package store_test

import (
	"fmt"
	"testing"

	"github.com/FreyreCorona/FluxCache/store"
)

func TestCRDTStore(t *testing.T) {
	testStore(t, func() store.Store { return store.NewCRDTStore() })
}

func TestCRDTStoreMerge(t *testing.T) {
	a := store.NewCRDTStore()
	b := store.NewCRDTStore()

	a.Set("a", "from-a")
	a.Set("b", "from-a")
	b.Set("b", "from-b")
	b.Set("c", "from-b")

	a.Merge(b)

	val, ok := a.Get("a")
	if !ok || val != "from-a" {
		t.Fatalf("expected 'from-a', got '%s'", val)
	}

	val, ok = a.Get("c")
	if !ok || val != "from-b" {
		t.Fatalf("expected 'from-b', got '%s'", val)
	}
}

func TestCRDTStoreSetWithTS(t *testing.T) {
	s := store.NewCRDTStore()

	s.SetWithTS("key", "first", 10)
	val, ok := s.Get("key")
	if !ok || val != "first" {
		t.Fatalf("expected 'first', got '%s'", val)
	}

	s.SetWithTS("key", "second", 5)
	val, _ = s.Get("key")
	if val != "first" {
		t.Fatalf("expected 'first' (higher ts wins), got '%s'", val)
	}

	s.SetWithTS("key", "third", 20)
	val, _ = s.Get("key")
	if val != "third" {
		t.Fatalf("expected 'third' (higher ts wins), got '%s'", val)
	}
}

func TestCRDTStoreSnapshot(t *testing.T) {
	s := store.NewCRDTStore()
	s.Set("k1", "v1")
	s.Set("k2", "v2")

	snap := s.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(snap))
	}
}

func BenchmarkCRDTStoreSet(b *testing.B) {
	s := store.NewCRDTStore()
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

func BenchmarkCRDTStoreGet(b *testing.B) {
	s := store.NewCRDTStore()
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

func BenchmarkCRDTStoreHSet(b *testing.B) {
	s := store.NewCRDTStore()
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

func BenchmarkCRDTStoreHGetAll(b *testing.B) {
	s := store.NewCRDTStore()
	s.HSet("benchhash", "a", "1")
	s.HSet("benchhash", "b", "2")
	s.HSet("benchhash", "c", "3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.HGetAll("benchhash")
	}
}

func BenchmarkCRDTStoreMerge(b *testing.B) {
	src := store.NewCRDTStore()
	dst := store.NewCRDTStore()
	for i := range 1000 {
		src.Set(fmt.Sprintf("key-a-%d", i), "val")
		dst.Set(fmt.Sprintf("key-b-%d", i), "val")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst.Merge(src)
	}
}
