package store

import (
	"hash/fnv"
	"maps"
	"sync/atomic"
)

type bucketData struct {
	strings map[string]string
	hashes  map[string]map[string]string
}

// LockFreeStore is a lock-free concurrent store using atomic CAS on buckets.
type LockFreeStore struct {
	buckets []atomic.Pointer[bucketData]
	mask    uint64
}

// NewLockFreeStore returns a new LockFreeStore with the given number of buckets.
func NewLockFreeStore(shardCount int) *LockFreeStore {
	if shardCount <= 0 {
		shardCount = 256
	}
	s := &LockFreeStore{
		buckets: make([]atomic.Pointer[bucketData], shardCount),
		mask:    uint64(shardCount - 1),
	}
	for i := range s.buckets {
		s.buckets[i].Store(&bucketData{
			strings: make(map[string]string),
			hashes:  make(map[string]map[string]string),
		})
	}
	return s
}

func (s *LockFreeStore) bucket(key string) *atomic.Pointer[bucketData] {
	h := fnv.New64a()
	h.Write([]byte(key))
	return &s.buckets[h.Sum64()&s.mask]
}

func clone[V any](m map[string]V) map[string]V {
	out := make(map[string]V, len(m))
	maps.Copy(out, m)
	return out
}

func cloneBucket(data *bucketData) *bucketData {
	strs := clone(data.strings)
	hshs := make(map[string]map[string]string, len(data.hashes))
	for h, inner := range data.hashes {
		hshs[h] = clone(inner)
	}
	return &bucketData{strings: strs, hashes: hshs}
}

func (s *LockFreeStore) Set(key, value string) {
	b := s.bucket(key)
	for {
		old := b.Load()
		data := cloneBucket(old)
		data.strings[key] = value
		if b.CompareAndSwap(old, data) {
			return
		}
	}
}

func (s *LockFreeStore) Del(key string) {
	b := s.bucket(key)
	for {
		old := b.Load()
		data := cloneBucket(old)
		delete(data.strings, key)
		delete(data.hashes, key)
		if b.CompareAndSwap(old, data) {
			return
		}
	}
}

func (s *LockFreeStore) Get(key string) (string, bool) {
	b := s.bucket(key)
	data := b.Load()
	v, ok := data.strings[key]
	return v, ok
}

func (s *LockFreeStore) HSet(hash, key, value string) {
	b := s.bucket(hash)
	for {
		old := b.Load()
		data := cloneBucket(old)
		if _, ok := data.hashes[hash]; !ok {
			data.hashes[hash] = make(map[string]string)
		}
		data.hashes[hash][key] = value
		if b.CompareAndSwap(old, data) {
			return
		}
	}
}

func (s *LockFreeStore) HGet(hash, key string) (string, bool) {
	b := s.bucket(hash)
	data := b.Load()
	inner, ok := data.hashes[hash]
	if !ok {
		return "", false
	}
	v, ok := inner[key]
	return v, ok
}

func (s *LockFreeStore) HGetAll(hash string) map[string]string {
	b := s.bucket(hash)
	data := b.Load()
	inner, ok := data.hashes[hash]
	if !ok {
		return nil
	}
	out := make(map[string]string, len(inner))
	maps.Copy(out, inner)
	return out
}

func (s *LockFreeStore) Close() error { return nil }
