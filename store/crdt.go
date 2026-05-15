package store

import "sync"

type crdtValue struct {
	value string
	ts    uint64
}

// CRDTStore is a CRDT-based store using last-writer-wins semantics with timestamps.
type CRDTStore struct {
	data   map[string]crdtValue
	hashes map[string]map[string]string
	clock  uint64
	mu     sync.RWMutex
}

// NewCRDTStore returns a new CRDTStore.
func NewCRDTStore() *CRDTStore {
	return &CRDTStore{
		data:   make(map[string]crdtValue),
		hashes: make(map[string]map[string]string),
	}
}

func (s *CRDTStore) nextTS() uint64 {
	s.clock++
	return s.clock
}

func (s *CRDTStore) Set(key, value string) {
	s.mu.Lock()
	ts := s.nextTS()
	s.data[key] = crdtValue{value: value, ts: ts}
	s.mu.Unlock()
}

func (s *CRDTStore) Del(key string) {
	s.mu.Lock()
	delete(s.data, key)
	delete(s.hashes, key)
	s.mu.Unlock()
}

func (s *CRDTStore) Get(key string) (string, bool) {
	s.mu.RLock()
	v, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	return v.value, true
}

// SetWithTS sets a key with the given timestamp (LWW CRDT semantics).
func (s *CRDTStore) SetWithTS(key, value string, ts uint64) {
	s.mu.Lock()
	existing, ok := s.data[key]
	if !ok || ts > existing.ts || (ts == existing.ts && false) {
		s.data[key] = crdtValue{value: value, ts: ts}
	}
	if ts > s.clock {
		s.clock = ts
	}
	s.mu.Unlock()
}

// Merge merges another CRDTStore into this one.
func (s *CRDTStore) Merge(other *CRDTStore) {
	other.mu.RLock()
	for key, v := range other.data {
		s.mu.Lock()
		existing, ok := s.data[key]
		if !ok || v.ts > existing.ts {
			s.data[key] = v
		}
		if v.ts > s.clock {
			s.clock = v.ts
		}
		s.mu.Unlock()
	}

	other.mu.RLock()
	for hash, inner := range other.hashes {
		s.mu.Lock()
		if _, ok := s.hashes[hash]; !ok {
			s.hashes[hash] = make(map[string]string)
		}
		for k, v := range inner {
			s.hashes[hash][k] = v
		}
		s.mu.Unlock()
	}
	other.mu.RUnlock()
}

// Snapshot returns a point-in-time copy of all key-value pairs.
func (s *CRDTStore) Snapshot() map[string]crdtValue {
	s.mu.RLock()
	out := make(map[string]crdtValue, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	s.mu.RUnlock()
	return out
}

func (s *CRDTStore) HSet(hash, key, value string) {
	s.mu.Lock()
	if _, ok := s.hashes[hash]; !ok {
		s.hashes[hash] = make(map[string]string)
	}
	s.hashes[hash][key] = value
	s.mu.Unlock()
}

func (s *CRDTStore) HGet(hash, key string) (string, bool) {
	s.mu.RLock()
	m, ok := s.hashes[hash]
	if !ok {
		s.mu.RUnlock()
		return "", false
	}
	v, ok := m[key]
	s.mu.RUnlock()
	return v, ok
}

func (s *CRDTStore) HGetAll(hash string) map[string]string {
	s.mu.RLock()
	m, ok := s.hashes[hash]
	if !ok {
		s.mu.RUnlock()
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	s.mu.RUnlock()
	return out
}

func (s *CRDTStore) Close() error { return nil }
