package evict

import (
	"sync"
	"time"
)

type LRUPolicy struct {
	mu     sync.Mutex
	access map[string]time.Time
}

func NewLRUPolicy() *LRUPolicy {
	return &LRUPolicy{access: make(map[string]time.Time)}
}

func (p *LRUPolicy) Name() string { return "allkeys-lru" }

func (p *LRUPolicy) Record(key string) {
	p.mu.Lock()
	p.access[key] = time.Now()
	p.mu.Unlock()
}

func (p *LRUPolicy) Delete(key string) {
	p.mu.Lock()
	delete(p.access, key)
	p.mu.Unlock()
}

func (p *LRUPolicy) Evict(candidates []string, _ map[string]time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(candidates) == 0 {
		return ""
	}

	oldest := candidates[0]
	oldestTime := p.access[oldest]

	for _, key := range candidates[1:] {
		if t, ok := p.access[key]; ok && t.Before(oldestTime) {
			oldest = key
			oldestTime = t
		}
	}
	return oldest
}
