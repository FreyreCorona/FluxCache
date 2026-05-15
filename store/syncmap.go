package store

import "sync"

type SyncMapStore struct {
	strings sync.Map
	hashes  sync.Map
}

func NewSyncMapStore() *SyncMapStore {
	return &SyncMapStore{}
}

func (s *SyncMapStore) Set(key, value string) {
	s.strings.Store(key, value)
}

func (s *SyncMapStore) Get(key string) (string, bool) {
	val, ok := s.strings.Load(key)
	if !ok {
		return "", false
	}
	str, _ := val.(string)
	return str, true
}

func (s *SyncMapStore) HSet(hash, key, value string) {
	h, _ := s.hashes.LoadOrStore(hash, &sync.Map{})
	h.(*sync.Map).Store(key, value)
}

func (s *SyncMapStore) HGet(hash, key string) (string, bool) {
	h, ok := s.hashes.Load(hash)
	if !ok {
		return "", false
	}
	val, ok := h.(*sync.Map).Load(key)
	if !ok {
		return "", false
	}
	str, _ := val.(string)
	return str, true
}

func (s *SyncMapStore) HGetAll(hash string) map[string]string {
	h, ok := s.hashes.Load(hash)
	if !ok {
		return nil
	}
	out := make(map[string]string)
	h.(*sync.Map).Range(func(key, val interface{}) bool {
		k, _ := key.(string)
		v, _ := val.(string)
		out[k] = v
		return true
	})
	return out
}

func (s *SyncMapStore) Close() error { return nil }
