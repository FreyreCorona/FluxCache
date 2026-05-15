package store_test

import (
	"os"
	"testing"

	"github.com/FreyreCorona/FluxCache/store"
)

func tempBitcask(t *testing.T) *store.BitcaskStore {
	t.Helper()
	f, err := os.CreateTemp("", "bitcask_*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	s, err := store.NewBitcaskStore(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.Close()
		os.Remove(f.Name())
	})
	return s
}

func TestBitcaskStoreSetGet(t *testing.T) {
	s := tempBitcask(t)

	s.Set("foo", "bar")
	val, ok := s.Get("foo")
	if !ok || val != "bar" {
		t.Fatalf("expected bar, got %s", val)
	}
}

func TestBitcaskStoreGetMissing(t *testing.T) {
	s := tempBitcask(t)

	_, ok := s.Get("missing")
	if ok {
		t.Fatal("expected false for missing key")
	}
}

func TestBitcaskStoreDel(t *testing.T) {
	s := tempBitcask(t)

	s.Set("foo", "bar")
	s.Del("foo")
	_, ok := s.Get("foo")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestBitcaskStoreHSetHGet(t *testing.T) {
	s := tempBitcask(t)

	s.HSet("hash", "field", "val")
	val, ok := s.HGet("hash", "field")
	if !ok || val != "val" {
		t.Fatalf("expected val, got %s", val)
	}
}

func TestBitcaskStoreHGetAll(t *testing.T) {
	s := tempBitcask(t)

	s.HSet("h", "a", "1")
	s.HSet("h", "b", "2")
	m := s.HGetAll("h")
	if len(m) != 2 || m["a"] != "1" || m["b"] != "2" {
		t.Fatalf("unexpected hash map: %v", m)
	}
}

func TestBitcaskStoreRecovery(t *testing.T) {
	f, err := os.CreateTemp("", "bitcask_recover_*.db")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()

	s1, err := store.NewBitcaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.Set("k1", "v1")
	s1.Set("k2", "v2")
	s1.HSet("h1", "f1", "val1")
	s1.Close()

	s2, err := store.NewBitcaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	defer os.Remove(path)

	val, ok := s2.Get("k1")
	if !ok || val != "v1" {
		t.Fatalf("expected v1 after recovery, got %s", val)
	}
	val, ok = s2.Get("k2")
	if !ok || val != "v2" {
		t.Fatalf("expected v2 after recovery, got %s", val)
	}
	val, ok = s2.HGet("h1", "f1")
	if !ok || val != "val1" {
		t.Fatalf("expected val1 after recovery, got %s", val)
	}
}

func TestBitcaskStoreRecoveryWithDelete(t *testing.T) {
	f, err := os.CreateTemp("", "bitcask_recover_del_*.db")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()

	s1, err := store.NewBitcaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.Set("k1", "v1")
	s1.Set("k2", "v2")
	s1.Del("k1")
	s1.Close()

	s2, err := store.NewBitcaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	defer os.Remove(path)

	_, ok := s2.Get("k1")
	if ok {
		t.Fatal("expected k1 to be deleted after recovery")
	}
	val, ok := s2.Get("k2")
	if !ok || val != "v2" {
		t.Fatalf("expected v2 after recovery, got %s", val)
	}
}
