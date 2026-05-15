package evict_test

import (
	"testing"
	"time"

	"github.com/FreyreCorona/FluxCache/evict"
	"github.com/FreyreCorona/FluxCache/store"
)


func TestTTLStore(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)

	ts.Set("k", "v")
	val, ok := ts.Get("k")
	if !ok || val != "v" {
		t.Fatalf("expected 'v', got '%s'", val)
	}
}

func TestTTLStoreExpiry(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)

	ts.SetWithTTL("k", "v", 50*time.Millisecond)
	val, ok := ts.Get("k")
	if !ok || val != "v" {
		t.Fatalf("expected 'v' before expiry")
	}

	time.Sleep(100 * time.Millisecond)
	_, ok = ts.Get("k")
	if ok {
		t.Fatal("expected key to be expired")
	}
}

func TestTTLStoreExpire(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)

	ts.Set("k", "v")
	if ok := ts.Expire("k", 50*time.Millisecond); !ok {
		t.Fatal("expected Expire to return true")
	}

	time.Sleep(100 * time.Millisecond)
	_, ok := ts.Get("k")
	if ok {
		t.Fatal("expected key to be expired after Expire")
	}
}

func TestTTLStoreExpireMissing(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)

	if ok := ts.Expire("nonexistent", time.Second); ok {
		t.Fatal("expected Expire on missing key to return false")
	}
}

func TestTTLStoreTTL(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)

	ts.Set("k", "v")
	if d := ts.TTL("k"); d != -2 {
		t.Fatalf("expected -2 for key without TTL, got %v", d)
	}

	ts.Expire("k", time.Second)
	d := ts.TTL("k")
	if d <= 0 || d > time.Second {
		t.Fatalf("expected TTL between 0 and 1s, got %v", d)
	}
}

func TestTTLStoreTTLExpired(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)

	ts.SetWithTTL("k", "v", 10*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	if d := ts.TTL("k"); d != -2 {
		t.Fatalf("expected -2 for expired key, got %v", d)
	}
}

func TestTTLStoreDel(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)

	ts.SetWithTTL("k", "v", time.Hour)
	ts.Del("k")
	if d := ts.TTL("k"); d != -2 {
		t.Fatal("expected -2 after Del")
	}
	_, ok := ts.Get("k")
	if ok {
		t.Fatal("expected missing after Del")
	}
}

func TestTTLStoreActiveSweep(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)

	ts.SetWithTTL("k1", "v1", 10*time.Millisecond)
	ts.SetWithTTL("k2", "v2", 10*time.Millisecond)

	time.Sleep(200 * time.Millisecond)

	_, ok := ts.Get("k1")
	if ok {
		t.Fatal("expected k1 to be swept")
	}
	_, ok = ts.Get("k2")
	if ok {
		t.Fatal("expected k2 to be swept")
	}
}

func TestTTLStoreHashExpiry(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)

	ts.SetWithTTL("h", "v", 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	_, ok := ts.HGet("h", "doesnt-matter")
	if ok {
		t.Fatal("expected hash HGet to be expired")
	}
}

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

	_, ok := ts.Get("b")
	if ok {
		t.Fatal("expected b to be evicted (LFU, freq=1)")
	}
}

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

func TestEvictionRandom(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)
	ts.SetEvictionPolicy(evict.NewRandomPolicy(), 2)

	ts.Set("a", "1")
	ts.Set("b", "2")
	_ = ts
}

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

func TestEvictionMaxKeysZero(t *testing.T) {
	inner := store.NewMapStore()
	ts := store.NewTTLStore(inner)
	ts.SetEvictionPolicy(evict.NewLRUPolicy(), 0)

	for i := 0; i < 100; i++ {
		ts.Set(string(rune('a'+i%26)), "v")
	}
	_ = ts
}
