package store

import (
	"maps"
	"math/rand/v2"
	"strings"
	"sync"
)

const (
	skipMaxLevel = 32
	skipP        = 0.25
)

type skipNode struct {
	key   string
	value string
	next  []*skipNode
}

// SkipListStore is a thread-safe store backed by a skip list with ordered iteration.
type SkipListStore struct {
	head   *skipNode
	level  int
	mu     sync.RWMutex
	hashes map[string]map[string]string
}

// NewSkipListStore returns a new SkipListStore.
func NewSkipListStore() *SkipListStore {
	return &SkipListStore{
		head:   &skipNode{next: make([]*skipNode, skipMaxLevel)},
		level:  1,
		hashes: make(map[string]map[string]string),
	}
}

func skipLevel() int {
	level := 1
	for level < skipMaxLevel && rand.Float64() < skipP {
		level++
	}
	return level
}

func (s *SkipListStore) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prev := make([]*skipNode, skipMaxLevel)
	curr := s.head
	for i := s.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].key < key {
			curr = curr.next[i]
		}
		prev[i] = curr
	}
	curr = curr.next[0]

	if curr != nil && curr.key == key {
		curr.value = value
		return
	}

	level := skipLevel()
	if level > s.level {
		for i := s.level; i < level; i++ {
			prev[i] = s.head
		}
		s.level = level
	}

	n := &skipNode{key: key, value: value, next: make([]*skipNode, level)}
	for i := range level {
		n.next[i] = prev[i].next[i]
		prev[i].next[i] = n
	}
}

func (s *SkipListStore) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prev := make([]*skipNode, skipMaxLevel)
	curr := s.head
	for i := s.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].key < key {
			curr = curr.next[i]
		}
		prev[i] = curr
	}
	curr = curr.next[0]
	if curr != nil && curr.key == key {
		for i := 0; i < len(curr.next); i++ {
			prev[i].next[i] = curr.next[i]
		}
	}
	delete(s.hashes, key)
}

func (s *SkipListStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	curr := s.head
	for i := s.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].key < key {
			curr = curr.next[i]
		}
	}
	curr = curr.next[0]

	if curr != nil && curr.key == key {
		return curr.value, true
	}
	return "", false
}

func (s *SkipListStore) HSet(hash, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.hashes[hash]; !ok {
		s.hashes[hash] = make(map[string]string)
	}
	s.hashes[hash][key] = value
}

func (s *SkipListStore) HGet(hash, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.hashes[hash]
	if !ok {
		return "", false
	}
	v, ok := m[key]
	return v, ok
}

func (s *SkipListStore) HGetAll(hash string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.hashes[hash]
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	maps.Copy(out, m)
	return out
}

func (s *SkipListStore) Close() error { return nil }

func (s *SkipListStore) PrefixKeys(prefix string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	curr := s.head
	for i := s.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].key < prefix {
			curr = curr.next[i]
		}
	}
	curr = curr.next[0]

	var out []string
	for curr != nil && strings.HasPrefix(curr.key, prefix) {
		out = append(out, curr.key)
		curr = curr.next[0]
	}
	return out
}

func (s *SkipListStore) RangeKeys(start, end string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	curr := s.head
	for i := s.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].key < start {
			curr = curr.next[i]
		}
	}
	curr = curr.next[0]

	var out []string
	for curr != nil && curr.key <= end {
		if curr.key >= start {
			out = append(out, curr.key)
		}
		curr = curr.next[0]
	}
	return out
}
