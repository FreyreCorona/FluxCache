package store

type Store interface {
	Set(key, value string)
	Get(key string) (string, bool)
	HSet(hash, key, value string)
	HGet(hash, key string) (string, bool)
	HGetAll(hash string) map[string]string
	Close() error
}
