package store_test

import (
	"sync"
	"testing"

	"github.com/FreyreCorona/FluxCache/store"
)

func testStore(t *testing.T, newStore func() store.Store) {
	t.Helper()

	t.Run("SetGet", func(t *testing.T) {
		s := newStore()
		s.Set("foo", "bar")
		val, ok := s.Get("foo")
		if !ok {
			t.Fatal("expected key to exist")
		}
		if val != "bar" {
			t.Fatalf("expected 'bar', got '%s'", val)
		}
	})

	t.Run("GetMissing", func(t *testing.T) {
		s := newStore()
		_, ok := s.Get("nonexistent")
		if ok {
			t.Fatal("expected missing key to return false")
		}
	})

	t.Run("SetOverwrite", func(t *testing.T) {
		s := newStore()
		s.Set("key", "old")
		s.Set("key", "new")
		val, ok := s.Get("key")
		if !ok {
			t.Fatal("expected key to exist")
		}
		if val != "new" {
			t.Fatalf("expected 'new', got '%s'", val)
		}
	})

	t.Run("Del", func(t *testing.T) {
		s := newStore()
		s.Set("tmp", "val")
		s.Del("tmp")
		_, ok := s.Get("tmp")
		if ok {
			t.Fatal("expected deleted key to return false")
		}
	})

	t.Run("DelHash", func(t *testing.T) {
		s := newStore()
		s.HSet("h", "f", "v")
		s.Del("h")
		_, ok := s.HGet("h", "f")
		if ok {
			t.Fatal("expected deleted hash to return false")
		}
	})

	t.Run("HSetHGet", func(t *testing.T) {
		s := newStore()
		s.HSet("hash1", "name", "alice")
		val, ok := s.HGet("hash1", "name")
		if !ok {
			t.Fatal("expected field to exist")
		}
		if val != "alice" {
			t.Fatalf("expected 'alice', got '%s'", val)
		}
	})

	t.Run("HGetMissingHash", func(t *testing.T) {
		s := newStore()
		_, ok := s.HGet("nonexistent", "key")
		if ok {
			t.Fatal("expected missing hash to return false")
		}
	})

	t.Run("HGetMissingField", func(t *testing.T) {
		s := newStore()
		s.HSet("h", "existing", "v")
		_, ok := s.HGet("h", "missing")
		if ok {
			t.Fatal("expected missing field to return false")
		}
	})

	t.Run("HGetAll", func(t *testing.T) {
		s := newStore()
		s.HSet("h", "a", "1")
		s.HSet("h", "b", "2")
		s.HSet("h", "c", "3")

		m := s.HGetAll("h")
		if len(m) != 3 {
			t.Fatalf("expected 3 fields, got %d", len(m))
		}
		if m["a"] != "1" || m["b"] != "2" || m["c"] != "3" {
			t.Fatalf("unexpected values: %v", m)
		}
	})

	t.Run("HGetAllMissing", func(t *testing.T) {
		s := newStore()
		m := s.HGetAll("nonexistent")
		if m != nil {
			t.Fatal("expected nil for missing hash")
		}
	})

	t.Run("ConcurrentSetGet", func(t *testing.T) {
		s := newStore()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := string(rune('a' + i%26))
				s.Set(key, "val")
				s.Get(key)
			}(i)
		}
		wg.Wait()
	})

	t.Run("ConcurrentHSetHGet", func(t *testing.T) {
		s := newStore()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				h := string(rune('a' + i%10))
				s.HSet(h, "field", "value")
				s.HGet(h, "field")
				s.HGetAll(h)
			}(i)
		}
		wg.Wait()
	})

	t.Run("Close", func(t *testing.T) {
		s := newStore()
		if err := s.Close(); err != nil {
			t.Fatalf("Close() returned error: %v", err)
		}
	})
}

func testOrderedStore(t *testing.T, newStore func() store.OrderedStore) {
	t.Helper()

	t.Run("PrefixKeys", func(t *testing.T) {
		s := newStore()
		s.Set("apple", "1")
		s.Set("application", "2")
		s.Set("appetite", "3")
		s.Set("banana", "4")

		keys := s.PrefixKeys("app")
		if len(keys) != 3 {
			t.Fatalf("expected 3 keys with prefix 'app', got %d: %v", len(keys), keys)
		}
	})

	t.Run("PrefixKeysNoMatch", func(t *testing.T) {
		s := newStore()
		s.Set("foo", "1")
		s.Set("bar", "2")

		keys := s.PrefixKeys("nonexistent")
		if len(keys) != 0 {
			t.Fatalf("expected 0 keys, got %d", len(keys))
		}
	})

	t.Run("RangeKeys", func(t *testing.T) {
		s := newStore()
		s.Set("a", "1")
		s.Set("b", "2")
		s.Set("c", "3")
		s.Set("d", "4")
		s.Set("e", "5")

		keys := s.RangeKeys("b", "d")
		if len(keys) != 3 {
			t.Fatalf("expected 3 keys in [b,d], got %d: %v", len(keys), keys)
		}
	})

	t.Run("RangeKeysEmpty", func(t *testing.T) {
		s := newStore()
		s.Set("a", "1")

		keys := s.RangeKeys("z", "zz")
		if len(keys) != 0 {
			t.Fatalf("expected 0 keys, got %d", len(keys))
		}
	})

	t.Run("RangeKeysAll", func(t *testing.T) {
		s := newStore()
		s.Set("k1", "v1")
		s.Set("k2", "v2")

		keys := s.RangeKeys("", "{")
		if len(keys) != 2 {
			t.Fatalf("expected 2 keys, got %d: %v", len(keys), keys)
		}
	})
}
