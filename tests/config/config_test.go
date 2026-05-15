package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FreyreCorona/FluxCache/config"
)

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
