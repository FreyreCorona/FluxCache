package persistence_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/FreyreCorona/FluxCache/persistence"
)

func TestRDBWriteReplay(t *testing.T) {
	rdb, cleanup := tempRDB(t)
	defer cleanup()

	cmds := []persistence.Command{
		{Name: "SET", Args: []string{"foo", "bar"}},
		{Name: "HSET", Args: []string{"hash1", "name", "alice"}},
		{Name: "SET", Args: []string{"baz", "qux"}},
	}
	for _, cmd := range cmds {
		if err := rdb.Write(cmd); err != nil {
			t.Fatalf("Write(%v) failed: %v", cmd, err)
		}
	}

	if err := rdb.Snapshot(); err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	var replayed []persistence.Command
	if err := rdb.Replay(func(cmd persistence.Command) {
		replayed = append(replayed, cmd)
	}); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if len(replayed) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(replayed))
	}
}

func TestRDBWriteOverwrite(t *testing.T) {
	rdb, cleanup := tempRDB(t)
	defer cleanup()

	rdb.Write(persistence.Command{Name: "SET", Args: []string{"key", "old"}})
	rdb.Write(persistence.Command{Name: "SET", Args: []string{"key", "new"}})
	rdb.Snapshot()

	var replayed []persistence.Command
	rdb.Replay(func(cmd persistence.Command) {
		replayed = append(replayed, cmd)
	})

	if len(replayed) != 1 {
		t.Fatalf("expected 1 command after overwrite, got %d", len(replayed))
	}
	if replayed[0].Args[1] != "new" {
		t.Fatalf("expected 'new', got '%s'", replayed[0].Args[1])
	}
}

func TestRDBEmptyReplay(t *testing.T) {
	rdb, cleanup := tempRDB(t)
	defer cleanup()

	count := 0
	if err := rdb.Replay(func(cmd persistence.Command) {
		count++
	}); err != nil {
		t.Fatalf("Replay on empty RDB failed: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 commands on empty RDB, got %d", count)
	}
}

func BenchmarkRDBWrite(b *testing.B) {
	f, err := os.CreateTemp("", "fluxcache-rdb-bench-*.rdb")
	if err != nil {
		b.Fatal(err)
	}
	_ = f.Close()
	defer os.Remove(f.Name())

	rdb, err := persistence.NewRDB(f.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer rdb.Close()

	cmds := make([]persistence.Command, b.N)
	for i := range cmds {
		cmds[i] = persistence.Command{
			Name: "SET",
			Args: []string{fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i)},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rdb.Write(cmds[i])
	}
}
