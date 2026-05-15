package evict

import "time"

// EvictionPolicy defines the interface for cache eviction strategies.
type EvictionPolicy interface {
	Name() string
	Record(key string)
	Delete(key string)
	Evict(candidates []string, expireAt map[string]time.Time) string
}
