package store

import (
	"sort"
	"strings"
	"sync"
)

type artLeaf struct {
	key   string
	value string
}

type artNode struct {
	prefix   []byte
	leaf     *artLeaf
	nodeType int
	keys     []byte
	children []*artNode
	index    [256]int16
}

const (
	_             = iota
	artNode4      = 4
	artNode16     = 16
	artNode48     = 48
	artNode256    = 256
	artEmptyIndex = -1
)

type ARTStore struct {
	root   *artNode
	mu     sync.RWMutex
	hashes map[string]map[string]string
}

func NewARTStore() *ARTStore {
	return &ARTStore{hashes: make(map[string]map[string]string)}
}

func (s *ARTStore) removeChild(n *artNode, b byte) {
	switch n.nodeType {
	case artNode4, artNode16:
		for i, k := range n.keys {
			if k == b {
				n.keys = append(n.keys[:i], n.keys[i+1:]...)
				n.children = append(n.children[:i], n.children[i+1:]...)
				return
			}
		}
	case artNode48:
		if idx := n.index[b]; idx != artEmptyIndex {
			n.index[b] = artEmptyIndex
		}
	case artNode256:
		n.children[b] = nil
	}
}

func (s *ARTStore) delNode(n *artNode, key string, depth int) *artNode {
	if n == nil {
		return nil
	}
	if n.leaf != nil {
		if n.leaf.key == key {
			return nil
		}
		return n
	}

	if n.prefix != nil {
		pxLen := len(n.prefix)
		if depth+pxLen > len(key) || key[depth:depth+pxLen] != string(n.prefix) {
			return n
		}
		depth += pxLen
	}

	var b byte
	if depth < len(key) {
		b = key[depth]
	}

	child := s.findChild(n, b)
	if child == nil {
		child = s.findChild(n, 0)
	}
	if child == nil {
		return n
	}

	if depth < len(key) {
		child = s.delNode(child, key, depth+1)
	} else {
		child = s.delNode(child, key, depth)
	}

	if child == nil {
		s.removeChild(n, b)
	}

	if len(n.children) == 0 && n.leaf == nil {
		return nil
	}
	return n
}

func (s *ARTStore) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.hashes, key)
	if s.root == nil {
		return
	}
	s.root = s.delNode(s.root, key, 0)
}

func (s *ARTStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.root == nil {
		return "", false
	}
	n := s.root
	depth := 0
	for n.leaf == nil {
		if n.prefix != nil {
			pxLen := len(n.prefix)
			if depth+pxLen > len(key) || key[depth:depth+pxLen] != string(n.prefix) {
				return "", false
			}
			depth += pxLen
		}
		if depth >= len(key) {
			child := s.findChild(n, 0)
			if child == nil {
				return "", false
			}
			n = child
			continue
		}
		n = s.findChild(n, key[depth])
		if n == nil {
			return "", false
		}
		depth++
	}
	if n.leaf.key == key {
		return n.leaf.value, true
	}
	return "", false
}

func (s *ARTStore) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.root == nil {
		s.root = &artNode{leaf: &artLeaf{key: key, value: value}}
		return
	}

	depth := 0
	n := s.root
	for n.leaf == nil {
		if n.prefix != nil {
			pxLen := len(n.prefix)
			if depth+pxLen > len(key) {
				s.splitPrefix(n, key, value)
				return
			}
			mismatch := 0
			for ; mismatch < pxLen && depth+mismatch < len(key) && n.prefix[mismatch] == key[depth+mismatch]; mismatch++ {
			}
			if mismatch < pxLen {
				s.splitPrefixAt(n, depth, mismatch, key, value)
				return
			}
			depth += pxLen
		}
		if depth >= len(key) {
			if n.leaf == nil {
				child := s.findChild(n, 0)
				if child != nil {
					if child.leaf.key == key {
						child.leaf.value = value
						return
					}
					s.addChild(n, 0, &artNode{leaf: &artLeaf{key: key, value: value}})
					return
				}
			}
			s.addChild(n, 0, &artNode{leaf: &artLeaf{key: key, value: value}})
			return
		}
		b := key[depth]
		child := s.findChild(n, b)
		if child == nil {
			s.addChild(n, b, &artNode{leaf: &artLeaf{key: key, value: value}})
			return
		}
		n = child
		depth++
	}

	if n.leaf.key == key {
		n.leaf.value = value
		return
	}

	oldKey := n.leaf.key
	oldValue := n.leaf.value

	newLeaf := &artNode{leaf: &artLeaf{key: key, value: value}}
	oldLeaf := &artNode{leaf: &artLeaf{key: oldKey, value: oldValue}}
	n.leaf = nil

	prefixLen := s.lcp(oldKey, key, depth)
	if prefixLen > 0 {
		n.prefix = []byte(oldKey[depth : depth+prefixLen])
	}

	childByte := byte(0)
	if depth+prefixLen < len(oldKey) {
		childByte = oldKey[depth+prefixLen]
	}
	newByte := byte(0)
	if depth+prefixLen < len(key) {
		newByte = key[depth+prefixLen]
	}

	n.nodeType = artNode4
	n.keys = make([]byte, 0, 4)
	n.children = make([]*artNode, 0, 4)
	s.addChild(n, childByte, oldLeaf)
	s.addChild(n, newByte, newLeaf)
}

func (s *ARTStore) splitPrefix(n *artNode, key, value string) {
	newNode := &artNode{
		nodeType: n.nodeType,
		keys:     n.keys,
		children: n.children,
		index:    n.index,
		leaf:     n.leaf,
	}
	if len(n.prefix) > 0 {
		newNode.prefix = make([]byte, len(n.prefix))
		copy(newNode.prefix, n.prefix)
	}
	n.leaf = nil
	n.nodeType = artNode4
	n.keys = make([]byte, 0, 4)
	n.children = make([]*artNode, 0, 4)
	n.prefix = nil
	s.addChild(n, newNode.prefix[0], newNode)
	n.prefix = make([]byte, 0)
	s.Set(key, value)
}

func (s *ARTStore) splitPrefixAt(n *artNode, depth, mismatch int, key, value string) {
	oldPrefix := n.prefix
	pxLen := len(oldPrefix)

	newNode := &artNode{
		nodeType: n.nodeType,
		keys:     n.keys,
		children: n.children,
		index:    n.index,
		leaf:     n.leaf,
	}
	if mismatch+1 < pxLen {
		newNode.prefix = make([]byte, pxLen-mismatch-1)
		copy(newNode.prefix, oldPrefix[mismatch+1:])
	}

	childByte := oldPrefix[mismatch]
	newByte := byte(0)
	if depth+mismatch < len(key) {
		newByte = key[depth+mismatch]
	}

	n.leaf = nil
	n.nodeType = artNode4
	n.keys = make([]byte, 0, 4)
	n.children = make([]*artNode, 0, 4)
	if mismatch > 0 {
		n.prefix = make([]byte, mismatch)
		copy(n.prefix, oldPrefix[:mismatch])
	} else {
		n.prefix = nil
	}

	s.addChild(n, childByte, newNode)
	s.addChild(n, newByte, &artNode{leaf: &artLeaf{key: key, value: value}})
}

func (s *ARTStore) findChild(n *artNode, b byte) *artNode {
	switch n.nodeType {
	case artNode4, artNode16:
		for i, k := range n.keys {
			if k == b {
				return n.children[i]
			}
		}
	case artNode48:
		if idx := n.index[b]; idx != artEmptyIndex {
			return n.children[idx]
		}
	case artNode256:
		return n.children[b]
	}
	return nil
}

func (s *ARTStore) addChild(n *artNode, b byte, child *artNode) {
	switch n.nodeType {
	case artNode4, artNode16:
		pos := 0
		for pos < len(n.keys) && n.keys[pos] < b {
			pos++
		}
		n.keys = append(n.keys, 0)
		copy(n.keys[pos+1:], n.keys[pos:])
		n.keys[pos] = b
		n.children = append(n.children, nil)
		copy(n.children[pos+1:], n.children[pos:])
		n.children[pos] = child

		if len(n.keys) >= cap(n.keys) && n.nodeType == artNode4 {
			s.grow4to16(n)
		} else if len(n.keys) >= cap(n.keys) && n.nodeType == artNode16 {
			s.grow16to48(n)
		}

	case artNode48:
		slot := len(n.children)
		n.index[b] = int16(slot)
		n.children = append(n.children, child)
		if len(n.children) == 48 {
			s.grow48to256(n)
		}

	case artNode256:
		n.children[b] = child
	}
}

func (s *ARTStore) grow4to16(n *artNode) {
	newKeys := make([]byte, len(n.keys), 16)
	copy(newKeys, n.keys)
	newChildren := make([]*artNode, len(n.children), 16)
	copy(newChildren, n.children)
	n.keys = newKeys
	n.children = newChildren
	n.nodeType = artNode16
}

func (s *ARTStore) grow16to48(n *artNode) {
	for i := range n.index {
		n.index[i] = artEmptyIndex
	}
	for i, k := range n.keys {
		n.index[k] = int16(i)
	}
	newChildren := make([]*artNode, len(n.children), 48)
	copy(newChildren, n.children)
	n.children = newChildren
	n.keys = nil
	n.nodeType = artNode48
}

func (s *ARTStore) grow48to256(n *artNode) {
	newChildren := make([]*artNode, 256)
	for i, child := range n.children {
		for k := range n.index {
			if n.index[k] == int16(i) {
				newChildren[k] = child
				break
			}
		}
	}
	n.children = newChildren
	n.index = [256]int16{}
	n.nodeType = artNode256
}

func (s *ARTStore) lcp(key1, key2 string, start int) int {
	limit := min(len(key1), len(key2)) - start
	if limit <= 0 {
		return 0
	}
	i := 0
	for i < limit && key1[start+i] == key2[start+i] {
		i++
	}
	return i
}

func (s *ARTStore) HSet(hash, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.hashes[hash]; !ok {
		s.hashes[hash] = make(map[string]string)
	}
	s.hashes[hash][key] = value
}

func (s *ARTStore) HGet(hash, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.hashes[hash]
	if !ok {
		return "", false
	}
	v, ok := m[key]
	return v, ok
}

func (s *ARTStore) HGetAll(hash string) map[string]string {
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

func (s *ARTStore) Close() error { return nil }

func (s *ARTStore) PrefixKeys(prefix string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.root == nil {
		return nil
	}

	n := s.root
	depth := 0
	for n.leaf == nil {
		if n.prefix != nil {
			pxLen := len(n.prefix)
			if depth+pxLen > len(prefix) {
				if string(n.prefix[:len(prefix)-depth]) == prefix[depth:] {
					return s.collectKeys(n)
				}
				return nil
			}
			if string(n.prefix) != prefix[depth:depth+pxLen] {
				return nil
			}
			depth += pxLen
		}
		if depth >= len(prefix) {
			return s.collectKeys(n)
		}
		n = s.findChild(n, prefix[depth])
		if n == nil {
			return nil
		}
		depth++
	}
	if strings.HasPrefix(n.leaf.key, prefix) {
		return []string{n.leaf.key}
	}
	return nil
}

func (s *ARTStore) RangeKeys(start, end string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []string
	s.collectRange(s.root, start, end, &out)
	sort.Strings(out)
	return out
}

func (s *ARTStore) collectKeys(n *artNode) []string {
	if n == nil {
		return nil
	}
	if n.leaf != nil {
		return []string{n.leaf.key}
	}
	var out []string
	switch n.nodeType {
	case artNode4, artNode16:
		for _, child := range n.children {
			out = append(out, s.collectKeys(child)...)
		}
	case artNode48:
		for i := range n.index {
			if n.index[i] != artEmptyIndex {
				out = append(out, s.collectKeys(n.children[n.index[i]])...)
			}
		}
	case artNode256:
		for _, child := range n.children {
			out = append(out, s.collectKeys(child)...)
		}
	}
	return out
}

func (s *ARTStore) collectRange(n *artNode, start, end string, out *[]string) {
	if n == nil {
		return
	}
	if n.leaf != nil {
		if n.leaf.key >= start && n.leaf.key <= end {
			*out = append(*out, n.leaf.key)
		}
		return
	}
	switch n.nodeType {
	case artNode4, artNode16:
		for _, child := range n.children {
			s.collectRange(child, start, end, out)
		}
	case artNode48:
		for i := range n.index {
			if n.index[i] != artEmptyIndex {
				s.collectRange(n.children[n.index[i]], start, end, out)
			}
		}
	case artNode256:
		for _, child := range n.children {
			s.collectRange(child, start, end, out)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
