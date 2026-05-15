package ttl

import (
	"math/rand/v2"
	"sync"
	"time"
)

type EvictionPolicy interface {
	Name() string
	Record(key string)
	Delete(key string)
	Evict(candidates []string, expireAt map[string]time.Time) string
}

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

type LFUPolicy struct {
	mu   sync.Mutex
	freq map[string]int
}

func NewLFUPolicy() *LFUPolicy {
	return &LFUPolicy{freq: make(map[string]int)}
}

func (p *LFUPolicy) Name() string { return "allkeys-lfu" }

func (p *LFUPolicy) Record(key string) {
	p.mu.Lock()
	p.freq[key]++
	p.mu.Unlock()
}

func (p *LFUPolicy) Delete(key string) {
	p.mu.Lock()
	delete(p.freq, key)
	p.mu.Unlock()
}

func (p *LFUPolicy) Evict(candidates []string, _ map[string]time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(candidates) == 0 {
		return ""
	}

	lowest := candidates[0]
	lowestFreq := p.freq[lowest]

	for _, key := range candidates[1:] {
		if f, ok := p.freq[key]; ok && f < lowestFreq {
			lowest = key
			lowestFreq = f
		}
	}
	return lowest
}

type TTLPolicy struct{}

func NewTTLPolicy() *TTLPolicy { return &TTLPolicy{} }

func (p *TTLPolicy) Name() string { return "volatile-ttl" }

func (p *TTLPolicy) Record(string) {}

func (p *TTLPolicy) Delete(string) {}

func (p *TTLPolicy) Evict(candidates []string, expireAt map[string]time.Time) string {
	if len(candidates) == 0 {
		return ""
	}

	now := time.Now()
	nearest := candidates[0]
	nearestTTL := time.Duration(0)
	if exp, ok := expireAt[nearest]; ok {
		nearestTTL = exp.Sub(now)
	}

	for _, key := range candidates[1:] {
		exp, ok := expireAt[key]
		if !ok {
			continue
		}
		ttl := exp.Sub(now)
		if ttl < nearestTTL {
			nearest = key
			nearestTTL = ttl
		}
	}
	return nearest
}

type RandomPolicy struct{}

func NewRandomPolicy() *RandomPolicy { return &RandomPolicy{} }

func (p *RandomPolicy) Name() string { return "allkeys-random" }

func (p *RandomPolicy) Record(string) {}

func (p *RandomPolicy) Delete(string) {}

func (p *RandomPolicy) Evict(candidates []string, _ map[string]time.Time) string {
	if len(candidates) == 0 {
		return ""
	}
	return candidates[rand.IntN(len(candidates))]
}

type NoEviction struct{}

func NewNoEviction() *NoEviction { return &NoEviction{} }

func (p *NoEviction) Name() string { return "none" }

func (p *NoEviction) Record(string) {}

func (p *NoEviction) Delete(string) {}

func (p *NoEviction) Evict([]string, map[string]time.Time) string { return "" }
