package config

import (
	"fmt"
	"os"
	"strconv"
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
	HealthPort int    `yaml:"health_port"`
	MaxConns   int    `yaml:"max_connections"`
	MaxMemory  string `yaml:"max_memory"`
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	SocketPath string `yaml:"socket_path"`
}

type StoreConfig struct {
	Type       string `yaml:"type"`
	File       string `yaml:"file"`
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

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Port:     6379,
			Network:  "tcp",
			MaxConns: 0,
		},
		Store: StoreConfig{
			Type: "map",
		},
		Persistence: PersistenceConfig{
			Type: "null",
		},
		Eviction: EvictionConfig{
			Policy:  "noeviction",
			MaxKeys: 0,
		},
	}
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

var (
	validNetworks     = []string{"tcp", "tls", "unix", "http", "grpc"}
	validStores       = []string{"map", "sharded", "syncmap", "lockfree", "skiplist", "bptree", "art", "crdt", "bitcask"}
	validPersistence  = []string{"null", "aof", "wal", "rdb", "dual"}
	validEvictions    = []string{"noeviction", "allkeys-lru", "allkeys-lfu", "allkeys-random", "volatile-ttl"}
)

func in(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("config: port %d out of range [1-65535]", c.Server.Port)
	}
	if c.Server.HealthPort < 0 || c.Server.HealthPort > 65535 {
		return fmt.Errorf("config: health_port %d out of range [0-65535]", c.Server.HealthPort)
	}
	if c.Server.MaxConns < 0 {
		return fmt.Errorf("config: max_connections must be >= 0")
	}
	if c.Server.MaxMemory != "" {
		if _, err := parseBytes(c.Server.MaxMemory); err != nil {
			return fmt.Errorf("config: invalid max_memory %q: %v", c.Server.MaxMemory, err)
		}
	}
	if !in(validNetworks, c.Server.Network) {
		return fmt.Errorf("config: unknown network %q", c.Server.Network)
	}
	if !in(validStores, c.Store.Type) {
		return fmt.Errorf("config: unknown store %q", c.Store.Type)
	}
	if !in(validPersistence, c.Persistence.Type) {
		return fmt.Errorf("config: unknown persistence %q", c.Persistence.Type)
	}
	if !in(validEvictions, c.Eviction.Policy) {
		return fmt.Errorf("config: unknown eviction policy %q", c.Eviction.Policy)
	}

	if c.Store.Type == "bitcask" && c.Store.File == "" {
		return fmt.Errorf("config: bitcask store requires file path")
	}

	if c.Server.Network == "tls" {
		if c.Server.CertFile == "" {
			return fmt.Errorf("config: tls network requires cert_file")
		}
		if c.Server.KeyFile == "" {
			return fmt.Errorf("config: tls network requires key_file")
		}
	}

	if c.Server.Network == "unix" && c.Server.SocketPath == "" {
		return fmt.Errorf("config: unix network requires socket_path")
	}

	if c.Persistence.Type == "dual" {
		if c.Persistence.Primary == nil {
			return fmt.Errorf("config: dual persistence requires primary")
		}
		if c.Persistence.Secondary == nil {
			return fmt.Errorf("config: dual persistence requires secondary")
		}
	}

	if c.Persistence.Type == "aof" || c.Persistence.Type == "wal" || c.Persistence.Type == "rdb" {
		if c.Persistence.File == "" {
			return fmt.Errorf("config: %s persistence requires file", c.Persistence.Type)
		}
	}

	return nil
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
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func buildStore(cfg StoreConfig) (store.Store, error) {
	switch cfg.Type {
	case "map":
		return store.NewMapStore(), nil
	case "sharded":
		n := cfg.ShardCount
		if n < 1 {
			n = 16
		}
		return store.NewShardedStore(n), nil
	case "syncmap":
		return store.NewSyncMapStore(), nil
	case "lockfree":
		n := cfg.ShardCount
		if n < 1 {
			n = 16
		}
		return store.NewLockFreeStore(n), nil
	case "skiplist":
		return store.NewSkipListStore(), nil
	case "bptree":
		return store.NewBPTreeStore(), nil
	case "art":
		return store.NewARTStore(), nil
	case "crdt":
		return store.NewCRDTStore(), nil
	case "bitcask":
		return store.NewBitcaskStore(cfg.File)
	default:
		return store.NewMapStore(), nil
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
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	inner, err := buildStore(cfg.Store)
	if err != nil {
		return nil, nil, err
	}
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
		return network.NewTCP(fmt.Sprintf(":%d", cfg.Port), cfg.MaxConns), nil
	case "tls":
		if cfg.CertFile == "" || cfg.KeyFile == "" {
			return nil, fmt.Errorf("config: tls requires cert_file and key_file")
		}
		return network.NewTLS(fmt.Sprintf(":%d", cfg.Port), cfg.CertFile, cfg.KeyFile, cfg.MaxConns), nil
	case "unix":
		if cfg.SocketPath == "" {
			return nil, fmt.Errorf("config: unix requires socket_path")
		}
		return network.NewUnix(cfg.SocketPath, cfg.MaxConns), nil
	case "http":
		return network.NewHTTP(fmt.Sprintf(":%d", cfg.Port)), nil
	case "grpc":
		return network.NewGRPC(fmt.Sprintf(":%d", cfg.Port)), nil
	default:
		return nil, fmt.Errorf("config: unknown network %q", cfg.Network)
	}
}

func (c *Config) MaxMemoryBytes() (int64, error) {
	return parseBytes(c.Server.MaxMemory)
}

func parseBytes(s string) (int64, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("too short")
	}
	var mult int64 = 1
	suffix := s[len(s)-2:]
	switch suffix {
	case "KB":
		mult = 1 << 10
	case "MB":
		mult = 1 << 20
	case "GB":
		mult = 1 << 30
	case "TB":
		mult = 1 << 40
	default:
		if s[len(s)-1] == 'B' {
			return 0, fmt.Errorf("unknown suffix")
		}
		mult = 1
		suffix = s[len(s)-1:]
		switch suffix {
		case "K":
			mult = 1 << 10
		case "M":
			mult = 1 << 20
		case "G":
			mult = 1 << 30
		case "T":
			mult = 1 << 40
		default:
			suffix = ""
		}
	}
	numStr := s[:len(s)-len(suffix)]
	if numStr == "" {
		return 0, fmt.Errorf("no numeric value")
	}
	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return n * mult, nil
}
