package evict

import "time"

type EvictionPolicy interface {
	Name() string
	Record(key string)
	Delete(key string)
	Evict(candidates []string, expireAt map[string]time.Time) string
}
