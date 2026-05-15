package evict

import (
	"math/rand/v2"
	"time"
)

// RandomPolicy implements eviction of a random key.
type RandomPolicy struct{}

// NewRandomPolicy creates a new RandomPolicy.
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
