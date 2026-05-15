package store

import (
	"sort"
	"strings"
	"sync"
)

const bpOrder = 32

type bpNode struct {
	keys     []string
	children []*bpNode
	values   []string
	next     *bpNode
}

func newLeaf() *bpNode {
	return &bpNode{keys: make([]string, 0, bpOrder), values: make([]string, 0, bpOrder)}
}

func newInternal() *bpNode {
	return &bpNode{keys: make([]string, 0, bpOrder), children: make([]*bpNode, 0, bpOrder+1)}
}

func (n *bpNode) isLeaf() bool { return n.values != nil }

// BPTreeStore is a thread-safe store backed by a B+ tree with ordered iteration.
type BPTreeStore struct {
	root   *bpNode
	mu     sync.RWMutex
	hashes map[string]map[string]string
}

// NewBPTreeStore returns a new BPTreeStore.
func NewBPTreeStore() *BPTreeStore {
	return &BPTreeStore{
		root:   newLeaf(),
		hashes: make(map[string]map[string]string),
	}
}

func (s *BPTreeStore) searchLeaf(key string) *bpNode {
	n := s.root
	for !n.isLeaf() {
		idx := sort.Search(len(n.keys), func(i int) bool { return n.keys[i] > key })
		n = n.children[idx]
	}
	return n
}

func (s *BPTreeStore) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	leaf := s.searchLeaf(key)
	idx := sort.Search(len(leaf.keys), func(i int) bool { return leaf.keys[i] >= key })
	if idx < len(leaf.keys) && leaf.keys[idx] == key {
		leaf.keys = append(leaf.keys[:idx], leaf.keys[idx+1:]...)
		leaf.values = append(leaf.values[:idx], leaf.values[idx+1:]...)
	}
	delete(s.hashes, key)
}

func (s *BPTreeStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	leaf := s.searchLeaf(key)
	idx := sort.Search(len(leaf.keys), func(i int) bool { return leaf.keys[i] >= key })
	if idx < len(leaf.keys) && leaf.keys[idx] == key {
		return leaf.values[idx], true
	}
	return "", false
}

func (s *BPTreeStore) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	leaf := s.searchLeaf(key)
	idx := sort.Search(len(leaf.keys), func(i int) bool { return leaf.keys[i] >= key })
	if idx < len(leaf.keys) && leaf.keys[idx] == key {
		leaf.values[idx] = value
		return
	}

	leaf.keys = insertAt(leaf.keys, idx, key)
	leaf.values = insertAt(leaf.values, idx, value)

	if len(leaf.keys) < bpOrder {
		return
	}

	s.split(leaf)
}

func (s *BPTreeStore) split(leaf *bpNode) {
	mid := bpOrder / 2

	newLeaf := newLeaf()
	newLeaf.keys = append(newLeaf.keys, leaf.keys[mid:]...)
	newLeaf.values = append(newLeaf.values, leaf.values[mid:]...)
	leaf.keys = leaf.keys[:mid]
	leaf.values = leaf.values[:mid]
	newLeaf.next = leaf.next
	leaf.next = newLeaf

	promoteKey := newLeaf.keys[0]
	s.insertIntoParent(leaf, promoteKey, newLeaf)
}

func (s *BPTreeStore) insertIntoParent(left *bpNode, key string, right *bpNode) {
	if s.root == left {
		newRoot := newInternal()
		newRoot.keys = append(newRoot.keys, key)
		newRoot.children = append(newRoot.children, left, right)
		s.root = newRoot
		return
	}

	parent, childIdx := s.findParent(s.root, left)

	parent.keys = insertAt(parent.keys, childIdx, key)
	parent.children = insertAt(parent.children, childIdx+1, right)

	if len(parent.keys) < bpOrder {
		return
	}

	s.splitInternal(parent)
}

func (s *BPTreeStore) findParent(n, target *bpNode) (*bpNode, int) {
	if n.isLeaf() {
		return nil, 0
	}
	for i, child := range n.children {
		if child == target {
			return n, i
		}
		if !child.isLeaf() {
			if p, idx := s.findParent(child, target); p != nil {
				return p, idx
			}
		}
	}
	return nil, 0
}

func (s *BPTreeStore) splitInternal(n *bpNode) {
	mid := bpOrder / 2
	promoteKey := n.keys[mid]

	newNode := newInternal()
	newNode.keys = append(newNode.keys, n.keys[mid+1:]...)
	newNode.children = append(newNode.children, n.children[mid+1:]...)
	n.keys = n.keys[:mid]
	n.children = n.children[:mid+1]

	if s.root == n {
		newRoot := newInternal()
		newRoot.keys = append(newRoot.keys, promoteKey)
		newRoot.children = append(newRoot.children, n, newNode)
		s.root = newRoot
		return
	}

	parent, childIdx := s.findParent(s.root, n)
	parent.keys = insertAt(parent.keys, childIdx, promoteKey)
	parent.children = insertAt(parent.children, childIdx+1, newNode)

	if len(parent.keys) >= bpOrder {
		s.splitInternal(parent)
	}
}

func (s *BPTreeStore) HSet(hash, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.hashes[hash]; !ok {
		s.hashes[hash] = make(map[string]string)
	}
	s.hashes[hash][key] = value
}

func (s *BPTreeStore) HGet(hash, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.hashes[hash]
	if !ok {
		return "", false
	}
	v, ok := m[key]
	return v, ok
}

func (s *BPTreeStore) HGetAll(hash string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.hashes[hash]
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (s *BPTreeStore) Close() error { return nil }

func (s *BPTreeStore) PrefixKeys(prefix string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	leaf := s.searchLeaf(prefix)
	var out []string
	for leaf != nil {
		for _, k := range leaf.keys {
			if strings.HasPrefix(k, prefix) {
				out = append(out, k)
			} else if k > prefix && !strings.HasPrefix(k, prefix) {
				return out
			}
		}
		leaf = leaf.next
	}
	return out
}

func (s *BPTreeStore) RangeKeys(start, end string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	leaf := s.searchLeaf(start)
	var out []string
	for leaf != nil {
		for _, k := range leaf.keys {
			if k >= start && k <= end {
				out = append(out, k)
			} else if k > end {
				return out
			}
		}
		leaf = leaf.next
	}
	return out
}

func insertAt[S ~[]E, E any](s S, idx int, v E) S {
	s = append(s, *new(E))
	copy(s[idx+1:], s[idx:])
	s[idx] = v
	return s
}
