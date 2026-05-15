package evict

import (
	"sync"
	"time"
)

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
