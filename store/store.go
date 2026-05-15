package store

// Store is a generic key-value and hash store.
type Store interface {
	Set(key, value string)
	Get(key string) (string, bool)
	Del(key string)
	HSet(hash, key, value string)
	HGet(hash, key string) (string, bool)
	HGetAll(hash string) map[string]string
	Close() error
}

// OrderedStore is a Store with ordered key iteration.
type OrderedStore interface {
	Store
	PrefixKeys(prefix string) []string
	RangeKeys(start, end string) []string
}
