package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FreyreCorona/FluxCache/config"
)

func TestFromEnvOverridesYAML(t *testing.T) {
	os.Setenv("FLUXCACHE_PORT", "9999")
	os.Setenv("FLUXCACHE_NETWORK", "http")
	os.Setenv("FLUXCACHE_STORE_TYPE", "sharded")
	os.Setenv("FLUXCACHE_SHARD_COUNT", "32")
	os.Setenv("FLUXCACHE_PERSISTENCE_TYPE", "aof")
	os.Setenv("FLUXCACHE_PERSISTENCE_FILE", "test.aof")
	os.Setenv("FLUXCACHE_EVICTION_POLICY", "allkeys-lru")
	os.Setenv("FLUXCACHE_MAX_KEYS", "5000")
	os.Setenv("FLUXCACHE_MAX_MEMORY", "256MB")
	os.Setenv("FLUXCACHE_HEALTH_PORT", "8080")
	os.Setenv("FLUXCACHE_MAX_CONNECTIONS", "100")
	os.Setenv("FLUXCACHE_CERT_FILE", "/certs/cert.pem")
	os.Setenv("FLUXCACHE_KEY_FILE", "/certs/key.pem")
	os.Setenv("FLUXCACHE_SOCKET_PATH", "/tmp/flux.sock")
	os.Setenv("FLUXCACHE_DEGREE", "8")
	os.Setenv("FLUXCACHE_PERSISTENCE_INTERVAL", "10s")

	defer func() {
		os.Unsetenv("FLUXCACHE_PORT")
		os.Unsetenv("FLUXCACHE_NETWORK")
		os.Unsetenv("FLUXCACHE_STORE_TYPE")
		os.Unsetenv("FLUXCACHE_SHARD_COUNT")
		os.Unsetenv("FLUXCACHE_PERSISTENCE_TYPE")
		os.Unsetenv("FLUXCACHE_PERSISTENCE_FILE")
		os.Unsetenv("FLUXCACHE_EVICTION_POLICY")
		os.Unsetenv("FLUXCACHE_MAX_KEYS")
		os.Unsetenv("FLUXCACHE_MAX_MEMORY")
		os.Unsetenv("FLUXCACHE_HEALTH_PORT")
		os.Unsetenv("FLUXCACHE_MAX_CONNECTIONS")
		os.Unsetenv("FLUXCACHE_CERT_FILE")
		os.Unsetenv("FLUXCACHE_KEY_FILE")
		os.Unsetenv("FLUXCACHE_SOCKET_PATH")
		os.Unsetenv("FLUXCACHE_DEGREE")
		os.Unsetenv("FLUXCACHE_PERSISTENCE_INTERVAL")
	}()

	path := filepath.Join(t.TempDir(), "test.yaml")
	original := config.Default()
	if err := original.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Server.Port != 9999 {
		t.Fatalf("expected port 9999, got %d", loaded.Server.Port)
	}
	if loaded.Server.Network != "http" {
		t.Fatalf("expected network http, got %s", loaded.Server.Network)
	}
	if loaded.Server.HealthPort != 8080 {
		t.Fatalf("expected health_port 8080, got %d", loaded.Server.HealthPort)
	}
	if loaded.Server.MaxConns != 100 {
		t.Fatalf("expected max_connections 100, got %d", loaded.Server.MaxConns)
	}
	if loaded.Server.MaxMemory != "256MB" {
		t.Fatalf("expected max_memory 256MB, got %s", loaded.Server.MaxMemory)
	}
	if loaded.Server.CertFile != "/certs/cert.pem" {
		t.Fatalf("expected cert_file /certs/cert.pem, got %s", loaded.Server.CertFile)
	}
	if loaded.Server.KeyFile != "/certs/key.pem" {
		t.Fatalf("expected key_file /certs/key.pem, got %s", loaded.Server.KeyFile)
	}
	if loaded.Server.SocketPath != "/tmp/flux.sock" {
		t.Fatalf("expected socket_path /tmp/flux.sock, got %s", loaded.Server.SocketPath)
	}
	if loaded.Store.Type != "sharded" {
		t.Fatalf("expected store sharded, got %s", loaded.Store.Type)
	}
	if loaded.Store.ShardCount != 32 {
		t.Fatalf("expected shard_count 32, got %d", loaded.Store.ShardCount)
	}
	if loaded.Store.Degree != 8 {
		t.Fatalf("expected degree 8, got %d", loaded.Store.Degree)
	}
	if loaded.Persistence.Type != "aof" {
		t.Fatalf("expected persistence aof, got %s", loaded.Persistence.Type)
	}
	if loaded.Persistence.File != "test.aof" {
		t.Fatalf("expected persistence file test.aof, got %s", loaded.Persistence.File)
	}
	if loaded.Persistence.Interval != "10s" {
		t.Fatalf("expected persistence interval 10s, got %s", loaded.Persistence.Interval)
	}
	if loaded.Eviction.Policy != "allkeys-lru" {
		t.Fatalf("expected eviction policy allkeys-lru, got %s", loaded.Eviction.Policy)
	}
	if loaded.Eviction.MaxKeys != 5000 {
		t.Fatalf("expected maxkeys 5000, got %d", loaded.Eviction.MaxKeys)
	}
}

func TestFromEnvDualPersistence(t *testing.T) {
	os.Setenv("FLUXCACHE_PERSISTENCE_PRIMARY_TYPE", "aof")
	os.Setenv("FLUXCACHE_PERSISTENCE_PRIMARY_FILE", "primary.aof")
	os.Setenv("FLUXCACHE_PERSISTENCE_SECONDARY_TYPE", "wal")
	os.Setenv("FLUXCACHE_PERSISTENCE_SECONDARY_FILE", "secondary.wal")

	defer func() {
		os.Unsetenv("FLUXCACHE_PERSISTENCE_PRIMARY_TYPE")
		os.Unsetenv("FLUXCACHE_PERSISTENCE_PRIMARY_FILE")
		os.Unsetenv("FLUXCACHE_PERSISTENCE_SECONDARY_TYPE")
		os.Unsetenv("FLUXCACHE_PERSISTENCE_SECONDARY_FILE")
	}()

	path := filepath.Join(t.TempDir(), "dual.yaml")
	cfg := config.Default()
	cfg.Persistence.Type = "dual"
	cfg.Persistence.Primary = &config.PersistenceConfig{Type: "aof", File: "p.aof"}
	cfg.Persistence.Secondary = &config.PersistenceConfig{Type: "wal", File: "s.wal"}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Persistence.Primary.Type != "aof" {
		t.Fatalf("expected primary type aof, got %s", loaded.Persistence.Primary.Type)
	}
	if loaded.Persistence.Primary.File != "primary.aof" {
		t.Fatalf("expected primary file primary.aof, got %s", loaded.Persistence.Primary.File)
	}
	if loaded.Persistence.Secondary.Type != "wal" {
		t.Fatalf("expected secondary type wal, got %s", loaded.Persistence.Secondary.Type)
	}
	if loaded.Persistence.Secondary.File != "secondary.wal" {
		t.Fatalf("expected secondary file secondary.wal, got %s", loaded.Persistence.Secondary.File)
	}
}

func TestFromEnvDoesNotOverrideUnset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "default.yaml")
	cfg := config.Default()
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Server.Port != 6379 {
		t.Fatalf("expected default port 6379, got %d", loaded.Server.Port)
	}
	if loaded.Store.Type != "map" {
		t.Fatalf("expected default store map, got %s", loaded.Store.Type)
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.yaml")

	original := config.Default()
	original.Server.Port = 9999
	original.Server.Network = "http"
	original.Store.Type = "sharded"
	original.Store.ShardCount = 32
	original.Persistence.Type = "aof"
	original.Persistence.File = "test.aof"
	original.Eviction.Policy = "allkeys-lru"
	original.Eviction.MaxKeys = 5000

	if err := original.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Server.Port != 9999 {
		t.Fatalf("expected port 9999, got %d", loaded.Server.Port)
	}
	if loaded.Server.Network != "http" {
		t.Fatalf("expected network http, got %s", loaded.Server.Network)
	}
	if loaded.Store.Type != "sharded" {
		t.Fatalf("expected store sharded, got %s", loaded.Store.Type)
	}
	if loaded.Store.ShardCount != 32 {
		t.Fatalf("expected shard_count 32, got %d", loaded.Store.ShardCount)
	}
	if loaded.Persistence.Type != "aof" {
		t.Fatalf("expected persistence aof, got %s", loaded.Persistence.Type)
	}
	if loaded.Persistence.File != "test.aof" {
		t.Fatalf("expected file test.aof, got %s", loaded.Persistence.File)
	}
	if loaded.Eviction.Policy != "allkeys-lru" {
		t.Fatalf("expected policy allkeys-lru, got %s", loaded.Eviction.Policy)
	}
	if loaded.Eviction.MaxKeys != 5000 {
		t.Fatalf("expected maxkeys 5000, got %d", loaded.Eviction.MaxKeys)
	}
}

func TestSaveDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "default.yaml")

	cfg := config.Default()
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Server.Port != 6379 {
		t.Fatalf("expected default port 6379, got %d", loaded.Server.Port)
	}
	if loaded.Server.Network != "tcp" {
		t.Fatalf("expected default network tcp, got %s", loaded.Server.Network)
	}
	_ = data
}

func TestValidatePortRange(t *testing.T) {
	cfg := config.Default()
	cfg.Server.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for port 0")
	}
	cfg.Server.Port = 70000
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for port 70000")
	}
}

func TestValidateInvalidNetwork(t *testing.T) {
	cfg := config.Default()
	cfg.Server.Network = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid network")
	}
}

func TestValidateTLSRequiresCertKey(t *testing.T) {
	cfg := config.Default()
	cfg.Server.Network = "tls"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error: tls requires cert and key")
	}
	cfg.Server.CertFile = "cert.pem"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error: tls requires key")
	}
	cfg.Server.KeyFile = "key.pem"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateUnixRequiresSocketPath(t *testing.T) {
	cfg := config.Default()
	cfg.Server.Network = "unix"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error: unix requires socket_path")
	}
	cfg.Server.SocketPath = "/tmp/test.sock"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBitcaskRequiresFile(t *testing.T) {
	cfg := config.Default()
	cfg.Store.Type = "bitcask"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error: bitcask requires file")
	}
	cfg.Store.File = "data.db"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDualPersistenceRequiresBoth(t *testing.T) {
	cfg := config.Default()
	cfg.Persistence.Type = "dual"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error: dual requires primary and secondary")
	}
	cfg.Persistence.Primary = &config.PersistenceConfig{Type: "aof", File: "p.aof"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error: dual requires secondary")
	}
	cfg.Persistence.Secondary = &config.PersistenceConfig{Type: "wal", File: "s.wal"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePersistenceRequiresFile(t *testing.T) {
	cfg := config.Default()
	cfg.Persistence.Type = "aof"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error: aof requires file")
	}
	cfg.Persistence.Type = "rdb"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error: rdb requires file")
	}
}

func TestValidateInvalidEviction(t *testing.T) {
	cfg := config.Default()
	cfg.Eviction.Policy = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid eviction policy")
	}
}

func TestValidateInvalidStore(t *testing.T) {
	cfg := config.Default()
	cfg.Store.Type = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid store type")
	}
}

func TestDualPersistenceSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dual.yaml")

	cfg := config.Default()
	cfg.Persistence.Type = "dual"
	cfg.Persistence.Primary = &config.PersistenceConfig{Type: "aof", File: "primary.aof"}
	cfg.Persistence.Secondary = &config.PersistenceConfig{Type: "wal", File: "secondary.wal"}

	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Persistence.Type != "dual" {
		t.Fatalf("expected dual, got %s", loaded.Persistence.Type)
	}
	if loaded.Persistence.Primary == nil || loaded.Persistence.Primary.Type != "aof" {
		t.Fatal("expected primary aof")
	}
	if loaded.Persistence.Secondary == nil || loaded.Persistence.Secondary.Type != "wal" {
		t.Fatal("expected secondary wal")
	}
}
