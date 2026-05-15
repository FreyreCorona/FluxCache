package evict

import "time"

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
