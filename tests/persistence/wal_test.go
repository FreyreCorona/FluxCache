package persistence_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/FreyreCorona/FluxCache/persistence"
)

func TestWALWriteReplay(t *testing.T) {
	wal, cleanup := tempWAL(t)
	testPersistence(t, wal, cleanup)
}

func TestWALEmptyReplay(t *testing.T) {
	wal, cleanup := tempWAL(t)
	defer cleanup()

	count := 0
	if err := wal.Replay(func(cmd persistence.Command) {
		count++
	}); err != nil {
		t.Fatalf("Replay on empty WAL failed: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 commands on empty WAL, got %d", count)
	}
}

func TestWALReplayMultipleOpenClose(t *testing.T) {
	f, err := os.CreateTemp("", "fluxcache-wal-*.wal")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	path := f.Name()
	defer os.Remove(path)

	cmds := []persistence.Command{
		{Name: "SET", Args: []string{"a", "1"}},
		{Name: "SET", Args: []string{"b", "2"}},
	}

	wal1, err := persistence.NewWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range cmds {
		if err := wal1.Write(cmd); err != nil {
			t.Fatal(err)
		}
	}
	wal1.Close()

	wal2, err := persistence.NewWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	defer wal2.Close()

	var replayed []persistence.Command
	wal2.Replay(func(cmd persistence.Command) {
		replayed = append(replayed, cmd)
	})

	if len(replayed) != 2 {
		t.Fatalf("expected 2 commands after reopen, got %d", len(replayed))
	}
}

func TestWALFlush(t *testing.T) {
	wal, cleanup := tempWAL(t)
	defer cleanup()

	cmd := persistence.Command{Name: "SET", Args: []string{"flush", "test"}}
	if err := wal.Write(cmd); err != nil {
		t.Fatal(err)
	}

	if err := wal.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	var replayed []persistence.Command
	wal.Replay(func(cmd persistence.Command) {
		replayed = append(replayed, cmd)
	})

	if len(replayed) != 1 {
		t.Fatalf("expected 1 command after flush, got %d", len(replayed))
	}
}

func BenchmarkWALWrite(b *testing.B) {
	f, err := os.CreateTemp("", "fluxcache-wal-bench-*.wal")
	if err != nil {
		b.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	wal, err := persistence.NewWAL(f.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer wal.Close()

	cmds := make([]persistence.Command, b.N)
	for i := range cmds {
		cmds[i] = persistence.Command{
			Name: "SET",
			Args: []string{fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i)},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wal.Write(cmds[i])
	}
}
