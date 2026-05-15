package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/FreyreCorona/FluxCache/evict"
	"github.com/FreyreCorona/FluxCache/network"
	"github.com/FreyreCorona/FluxCache/persistence"
	"github.com/FreyreCorona/FluxCache/store"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Store       StoreConfig       `yaml:"store"`
	Persistence PersistenceConfig `yaml:"persistence"`
	Eviction    EvictionConfig    `yaml:"eviction"`
}

type ServerConfig struct {
	Port       int    `yaml:"port"`
	Network    string `yaml:"network"`
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	SocketPath string `yaml:"socket_path"`
}

type StoreConfig struct {
	Type       string `yaml:"type"`
	ShardCount int    `yaml:"shard_count"`
	Degree     int    `yaml:"degree"`
}

type PersistenceConfig struct {
	Type      string             `yaml:"type"`
	File      string             `yaml:"file"`
	Interval  string             `yaml:"interval"`
	Primary   *PersistenceConfig `yaml:"primary"`
	Secondary *PersistenceConfig `yaml:"secondary"`
}

type EvictionConfig struct {
	Policy  string `yaml:"policy"`
	MaxKeys int    `yaml:"maxkeys"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 6379
	}
	if cfg.Server.Network == "" {
		cfg.Server.Network = "tcp"
	}
	if cfg.Store.Type == "" {
		cfg.Store.Type = "map"
	}
	if cfg.Persistence.Type == "" {
		cfg.Persistence.Type = "null"
	}
	if cfg.Eviction.Policy == "" {
		cfg.Eviction.Policy = "noeviction"
	}
	return &cfg, nil
}

func buildStore(cfg StoreConfig) store.Store {
	switch cfg.Type {
	case "map":
		return store.NewMapStore()
	case "sharded":
		n := cfg.ShardCount
		if n < 1 {
			n = 16
		}
		return store.NewShardedStore(n)
	case "syncmap":
		return store.NewSyncMapStore()
	case "lockfree":
		n := cfg.ShardCount
		if n < 1 {
			n = 16
		}
		return store.NewLockFreeStore(n)
	case "skiplist":
		return store.NewSkipListStore()
	case "bptree":
		return store.NewBPTreeStore()
	case "art":
		return store.NewARTStore()
	case "crdt":
		return store.NewCRDTStore()
	default:
		return store.NewMapStore()
	}
}

func buildPersistence(cfg PersistenceConfig) (persistence.Persistence, error) {
	switch cfg.Type {
	case "aof":
		return persistence.NewAOF(cfg.File)
	case "wal":
		return persistence.NewWAL(cfg.File)
	case "rdb":
		if cfg.Interval != "" {
			d, err := time.ParseDuration(cfg.Interval)
			if err != nil {
				return nil, fmt.Errorf("config: rdb: invalid interval %q: %w", cfg.Interval, err)
			}
			return persistence.NewRDBWithInterval(cfg.File, d)
		}
		return persistence.NewRDB(cfg.File)
	case "dual":
		if cfg.Primary == nil || cfg.Secondary == nil {
			return nil, fmt.Errorf("config: dual persistence requires primary and secondary")
		}
		primary, err := buildPersistence(*cfg.Primary)
		if err != nil {
			return nil, fmt.Errorf("config: dual primary: %w", err)
		}
		secondary, err := buildPersistence(*cfg.Secondary)
		if err != nil {
			return nil, fmt.Errorf("config: dual secondary: %w", err)
		}
		return persistence.NewDualPersistence(primary, secondary), nil
	case "null":
		return persistence.NewNullPersistence(), nil
	default:
		return persistence.NewNullPersistence(), nil
	}
}

func buildEvictionPolicy(cfg EvictionConfig) (evict.EvictionPolicy, error) {
	switch cfg.Policy {
	case "allkeys-lru":
		return evict.NewLRUPolicy(), nil
	case "allkeys-lfu":
		return evict.NewLFUPolicy(), nil
	case "allkeys-random":
		return evict.NewRandomPolicy(), nil
	case "volatile-ttl":
		return evict.NewTTLPolicy(), nil
	case "noeviction":
		return evict.NewNoEviction(), nil
	default:
		return nil, fmt.Errorf("config: unknown eviction policy %q", cfg.Policy)
	}
}

func Build(cfg *Config) (*store.TTLStore, persistence.Persistence, error) {
	inner := buildStore(cfg.Store)
	ts := store.NewTTLStore(inner)

	policy, err := buildEvictionPolicy(cfg.Eviction)
	if err != nil {
		ts.Close()
		return nil, nil, err
	}
	ts.SetEvictionPolicy(policy, cfg.Eviction.MaxKeys)

	p, err := buildPersistence(cfg.Persistence)
	if err != nil {
		ts.Close()
		return nil, nil, err
	}

	return ts, p, nil
}

func BuildNetwork(cfg ServerConfig) (network.Network, error) {
	switch cfg.Network {
	case "tcp":
		return network.NewTCP(fmt.Sprintf(":%d", cfg.Port)), nil
	case "tls":
		if cfg.CertFile == "" || cfg.KeyFile == "" {
			return nil, fmt.Errorf("config: tls requires cert_file and key_file")
		}
		return network.NewTLS(fmt.Sprintf(":%d", cfg.Port), cfg.CertFile, cfg.KeyFile), nil
	case "unix":
		if cfg.SocketPath == "" {
			return nil, fmt.Errorf("config: unix requires socket_path")
		}
		return network.NewUnix(cfg.SocketPath), nil
	case "http":
		return network.NewHTTP(fmt.Sprintf(":%d", cfg.Port)), nil
	default:
		return nil, fmt.Errorf("config: unknown network %q", cfg.Network)
	}
}
