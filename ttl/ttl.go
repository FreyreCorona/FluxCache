package ttl

import (
	"sync"
	"time"

	"github.com/FreyreCorona/FluxCache/store"
)

type TTLStore struct {
	inner    store.Store
	mu       sync.Mutex
	expireAt map[string]time.Time
	done     chan struct{}
}

func NewTTLStore(inner store.Store) *TTLStore {
	t := &TTLStore{
		inner:    inner,
		expireAt: make(map[string]time.Time),
		done:     make(chan struct{}),
	}
	go t.sweepLoop()
	return t
}

func (t *TTLStore) isExpired(key string) bool {
	t.mu.Lock()
	exp, ok := t.expireAt[key]
	t.mu.Unlock()
	return ok && time.Now().After(exp)
}

func (t *TTLStore) delExpired(key string) {
	t.mu.Lock()
	delete(t.expireAt, key)
	t.mu.Unlock()
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
		}
	}
	t.mu.Unlock()
	for _, key := range expired {
		t.inner.Del(key)
	}
}

func (t *TTLStore) Set(key, value string) {
	t.inner.Set(key, value)
}

func (t *TTLStore) SetWithTTL(key, value string, ttl time.Duration) {
	t.mu.Lock()
	t.expireAt[key] = time.Now().Add(ttl)
	t.mu.Unlock()
	t.inner.Set(key, value)
}

func (t *TTLStore) Get(key string) (string, bool) {
	if t.isExpired(key) {
		t.delExpired(key)
		return "", false
	}
	return t.inner.Get(key)
}

func (t *TTLStore) Del(key string) {
	t.mu.Lock()
	delete(t.expireAt, key)
	t.mu.Unlock()
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
}

func (t *TTLStore) HGet(hash, key string) (string, bool) {
	if t.isExpired(hash) {
		t.delExpired(hash)
		return "", false
	}
	return t.inner.HGet(hash, key)
}

func (t *TTLStore) HGetAll(hash string) map[string]string {
	if t.isExpired(hash) {
		t.delExpired(hash)
		return nil
	}
	return t.inner.HGetAll(hash)
}

func (t *TTLStore) Close() error {
	close(t.done)
	return t.inner.Close()
}
