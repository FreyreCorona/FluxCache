package persistence_test

import (
	"os"
	"testing"

	"github.com/FreyreCorona/FluxCache/persistence"
)

func tempAOF(t *testing.T) (*persistence.AOF, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "fluxcache-aof-*.aof")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	aof, err := persistence.NewAOF(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatal(err)
	}

	return aof, func() {
		aof.Close()
		os.Remove(f.Name())
	}
}

func tempWAL(t *testing.T) (*persistence.WAL, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "fluxcache-wal-*.wal")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	wal, err := persistence.NewWAL(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatal(err)
	}

	return wal, func() {
		wal.Close()
		os.Remove(f.Name())
	}
}

func tempRDB(t *testing.T) (*persistence.RDB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "fluxcache-rdb-*.rdb")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	rdb, err := persistence.NewRDB(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatal(err)
	}

	return rdb, func() {
		rdb.Close()
		os.Remove(f.Name())
	}
}

func testPersistence(t *testing.T, p persistence.Persistence, cleanup func()) {
	t.Helper()
	defer cleanup()

	cmds := []persistence.Command{
		{Name: "SET", Args: []string{"foo", "bar"}},
		{Name: "HSET", Args: []string{"hash1", "name", "alice"}},
		{Name: "SET", Args: []string{"baz", "qux"}},
		{Name: "HSET", Args: []string{"hash1", "age", "30"}},
	}

	for _, cmd := range cmds {
		if err := p.Write(cmd); err != nil {
			t.Fatalf("Write(%v) failed: %v", cmd, err)
		}
	}

	var replayed []persistence.Command
	if err := p.Replay(func(cmd persistence.Command) {
		replayed = append(replayed, cmd)
	}); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if len(replayed) != len(cmds) {
		t.Fatalf("expected %d commands, got %d", len(cmds), len(replayed))
	}

	for i, cmd := range cmds {
		if replayed[i].Name != cmd.Name {
			t.Fatalf("cmd %d: expected Name '%s', got '%s'", i, cmd.Name, replayed[i].Name)
		}
		if len(replayed[i].Args) != len(cmd.Args) {
			t.Fatalf("cmd %d: expected %d args, got %d", i, len(cmd.Args), len(replayed[i].Args))
		}
		for j := range cmd.Args {
			if replayed[i].Args[j] != cmd.Args[j] {
				t.Fatalf("cmd %d arg %d: expected '%s', got '%s'", i, j, cmd.Args[j], replayed[i].Args[j])
			}
		}
	}
}
