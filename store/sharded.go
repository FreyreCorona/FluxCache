package store

import (
	"hash/fnv"
	"maps"
	"sync"
)

type shard struct {
	mu      sync.RWMutex
	strings map[string]string
	hashes  map[string]map[string]string
}

// ShardedStore is a sharded in-memory store that reduces lock contention.
type ShardedStore struct {
	shards []*shard
	mask   uint64
}

// NewShardedStore returns a new ShardedStore with the given number of shards.
func NewShardedStore(shardCount int) *ShardedStore {
	if shardCount <= 0 {
		shardCount = 256
	}
	shards := make([]*shard, shardCount)
	for i := range shards {
		shards[i] = &shard{
			strings: make(map[string]string),
			hashes:  make(map[string]map[string]string),
		}
	}
	return &ShardedStore{
		shards: shards,
		mask:   uint64(shardCount - 1),
	}
}

func (s *ShardedStore) getShard(key string) *shard {
	h := fnv.New64a()
	h.Write([]byte(key))
	return s.shards[h.Sum64()&s.mask]
}

func (s *ShardedStore) Set(key, value string) {
	sh := s.getShard(key)
	sh.mu.Lock()
	sh.strings[key] = value
	sh.mu.Unlock()
}

func (s *ShardedStore) Del(key string) {
	sh := s.getShard(key)
	sh.mu.Lock()
	delete(sh.strings, key)
	delete(sh.hashes, key)
	sh.mu.Unlock()
}

func (s *ShardedStore) Get(key string) (string, bool) {
	sh := s.getShard(key)
	sh.mu.RLock()
	val, ok := sh.strings[key]
	sh.mu.RUnlock()
	return val, ok
}

func (s *ShardedStore) HSet(hash, key, value string) {
	sh := s.getShard(hash)
	sh.mu.Lock()
	if _, ok := sh.hashes[hash]; !ok {
		sh.hashes[hash] = make(map[string]string)
	}
	sh.hashes[hash][key] = value
	sh.mu.Unlock()
}

func (s *ShardedStore) HGet(hash, key string) (string, bool) {
	sh := s.getShard(hash)
	sh.mu.RLock()
	m, ok := sh.hashes[hash]
	if !ok {
		sh.mu.RUnlock()
		return "", false
	}
	val, ok := m[key]
	sh.mu.RUnlock()
	return val, ok
}

func (s *ShardedStore) HGetAll(hash string) map[string]string {
	sh := s.getShard(hash)
	sh.mu.RLock()
	m, ok := sh.hashes[hash]
	if !ok {
		sh.mu.RUnlock()
		return nil
	}
	out := make(map[string]string, len(m))
	maps.Copy(out, m)
	sh.mu.RUnlock()
	return out
}

func (s *ShardedStore) Close() error { return nil }
