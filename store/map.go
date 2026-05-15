package store

import "maps"

import "sync"

// MapStore is a thread-safe in-memory store backed by Go maps.
type MapStore struct {
	strings map[string]string
	hashes  map[string]map[string]string
	mu      sync.RWMutex
}

// NewMapStore returns a new MapStore.
func NewMapStore() *MapStore {
	return &MapStore{
		strings: make(map[string]string),
		hashes:  make(map[string]map[string]string),
	}
}

func (s *MapStore) Set(key, value string) {
	s.mu.Lock()
	s.strings[key] = value
	s.mu.Unlock()
}

func (s *MapStore) Del(key string) {
	s.mu.Lock()
	delete(s.strings, key)
	delete(s.hashes, key)
	s.mu.Unlock()
}

func (s *MapStore) Get(key string) (string, bool) {
	s.mu.RLock()
	val, ok := s.strings[key]
	s.mu.RUnlock()
	return val, ok
}

func (s *MapStore) HSet(hash, key, value string) {
	s.mu.Lock()
	if _, ok := s.hashes[hash]; !ok {
		s.hashes[hash] = make(map[string]string)
	}
	s.hashes[hash][key] = value
	s.mu.Unlock()
}

func (s *MapStore) HGet(hash, key string) (string, bool) {
	s.mu.RLock()
	m, ok := s.hashes[hash]
	if !ok {
		s.mu.RUnlock()
		return "", false
	}
	val, ok := m[key]
	s.mu.RUnlock()
	return val, ok
}

func (s *MapStore) HGetAll(hash string) map[string]string {
	s.mu.RLock()
	m, ok := s.hashes[hash]
	if !ok {
		s.mu.RUnlock()
		return nil
	}
	out := make(map[string]string, len(m))
	maps.Copy(out, m)
	s.mu.RUnlock()
	return out
}

func (s *MapStore) Close() error { return nil }
