package store

import (
	"sync"
	"time"

	"github.com/FreyreCorona/FluxCache/evict"
)

type TTLStore struct {
	inner    Store
	mu       sync.Mutex
	expireAt map[string]time.Time
	allKeys  map[string]struct{}
	policy   evict.EvictionPolicy
	maxKeys  int
	done     chan struct{}
}

func NewTTLStore(inner Store) *TTLStore {
	t := &TTLStore{
		inner:    inner,
		expireAt: make(map[string]time.Time),
		allKeys:  make(map[string]struct{}),
		policy:   &evict.NoEviction{},
		done:     make(chan struct{}),
	}
	go t.sweepLoop()
	return t
}

func (t *TTLStore) SetEvictionPolicy(p evict.EvictionPolicy, maxKeys int) {
	t.mu.Lock()
	t.policy = p
	t.maxKeys = maxKeys
	t.mu.Unlock()
}

func (t *TTLStore) track(key string) {
	t.mu.Lock()
	t.allKeys[key] = struct{}{}
	t.mu.Unlock()
	t.policy.Record(key)
}

func (t *TTLStore) untrack(key string) {
	t.mu.Lock()
	delete(t.allKeys, key)
	delete(t.expireAt, key)
	t.mu.Unlock()
	t.policy.Delete(key)
}

func (t *TTLStore) candidates() []string {
	t.mu.Lock()
	name := t.policy.Name()
	isVolatile := len(name) > 8 && name[:8] == "volatile"
	keys := make([]string, 0)
	if isVolatile {
		for k := range t.expireAt {
			keys = append(keys, k)
		}
	} else {
		for k := range t.allKeys {
			keys = append(keys, k)
		}
	}
	t.mu.Unlock()
	return keys
}

func (t *TTLStore) evictIfNeeded() {
	t.mu.Lock()
	shouldEvict := t.maxKeys > 0 && len(t.allKeys) > t.maxKeys
	t.mu.Unlock()
	if !shouldEvict {
		return
	}

	candidates := t.candidates()
	if len(candidates) == 0 {
		return
	}

	t.mu.Lock()
	key := t.policy.Evict(candidates, t.expireAt)
	t.mu.Unlock()
	if key == "" {
		return
	}
	t.untrack(key)
	t.inner.Del(key)
}

func (t *TTLStore) isExpired(key string) bool {
	t.mu.Lock()
	exp, ok := t.expireAt[key]
	t.mu.Unlock()
	return ok && time.Now().After(exp)
}

func (t *TTLStore) delExpired(key string) {
	t.untrack(key)
	t.inner.Del(key)
}

func (t *TTLStore) sweepLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.sweep()
		case <-t.done:
			return
		}
	}
}

func (t *TTLStore) sweep() {
	t.mu.Lock()
	now := time.Now()
	expired := make([]string, 0)
	for key, exp := range t.expireAt {
		if now.After(exp) {
			expired = append(expired, key)
			delete(t.expireAt, key)
			delete(t.allKeys, key)
		}
	}
	t.mu.Unlock()
	for _, key := range expired {
		t.policy.Delete(key)
		t.inner.Del(key)
	}
}

func (t *TTLStore) Set(key, value string) {
	t.inner.Set(key, value)
	t.track(key)
	t.evictIfNeeded()
}

func (t *TTLStore) SetWithTTL(key, value string, ttl time.Duration) {
	t.mu.Lock()
	t.expireAt[key] = time.Now().Add(ttl)
	t.mu.Unlock()
	t.Set(key, value)
}

func (t *TTLStore) Get(key string) (string, bool) {
	if t.isExpired(key) {
		t.delExpired(key)
		return "", false
	}
	t.policy.Record(key)
	return t.inner.Get(key)
}

func (t *TTLStore) Del(key string) {
	t.untrack(key)
	t.inner.Del(key)
}

func (t *TTLStore) Expire(key string, ttl time.Duration) bool {
	_, ok := t.inner.Get(key)
	if !ok {
		return false
	}
	t.mu.Lock()
	t.expireAt[key] = time.Now().Add(ttl)
	t.mu.Unlock()
	return true
}

func (t *TTLStore) TTL(key string) time.Duration {
	t.mu.Lock()
	exp, ok := t.expireAt[key]
	t.mu.Unlock()
	if !ok {
		return -2
	}
	d := time.Until(exp)
	if d <= 0 {
		return -2
	}
	return d
}

func (t *TTLStore) HSet(hash, key, value string) {
	t.inner.HSet(hash, key, value)
	t.track(hash)
	t.evictIfNeeded()
}

func (t *TTLStore) HGet(hash, key string) (string, bool) {
	if t.isExpired(hash) {
		t.delExpired(hash)
		return "", false
	}
	t.policy.Record(hash)
	return t.inner.HGet(hash, key)
}

func (t *TTLStore) HGetAll(hash string) map[string]string {
	if t.isExpired(hash) {
		t.delExpired(hash)
		return nil
	}
	t.policy.Record(hash)
	return t.inner.HGetAll(hash)
}

func (t *TTLStore) Close() error {
	close(t.done)
	return t.inner.Close()
}
