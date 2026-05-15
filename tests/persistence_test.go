package tests

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

func TestAOFWriteReplay(t *testing.T) {
	aof, cleanup := tempAOF(t)
	defer cleanup()

	cmds := []persistence.Command{
		{Name: "SET", Args: []string{"foo", "bar"}},
		{Name: "HSET", Args: []string{"hash1", "name", "alice"}},
		{Name: "SET", Args: []string{"baz", "qux"}},
		{Name: "HSET", Args: []string{"hash1", "age", "30"}},
	}

	for _, cmd := range cmds {
		if err := aof.Write(cmd); err != nil {
			t.Fatalf("Write(%v) failed: %v", cmd, err)
		}
	}

	var replayed []persistence.Command
	if err := aof.Replay(func(cmd persistence.Command) {
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

func TestAOFEmptyReplay(t *testing.T) {
	aof, cleanup := tempAOF(t)
	defer cleanup()

	count := 0
	if err := aof.Replay(func(cmd persistence.Command) {
		count++
	}); err != nil {
		t.Fatalf("Replay on empty AOF failed: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 commands on empty AOF, got %d", count)
	}
}

func TestAOFReplayMultipleOpenClose(t *testing.T) {
	path := tempFile(t)
	defer os.Remove(path)

	cmds := []persistence.Command{
		{Name: "SET", Args: []string{"a", "1"}},
		{Name: "SET", Args: []string{"b", "2"}},
	}

	aof1, err := persistence.NewAOF(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range cmds {
		if err := aof1.Write(cmd); err != nil {
			t.Fatal(err)
		}
	}
	aof1.Close()

	aof2, err := persistence.NewAOF(path)
	if err != nil {
		t.Fatal(err)
	}
	defer aof2.Close()

	var replayed []persistence.Command
	aof2.Replay(func(cmd persistence.Command) {
		replayed = append(replayed, cmd)
	})

	if len(replayed) != 2 {
		t.Fatalf("expected 2 commands after reopen, got %d", len(replayed))
	}
}

func tempFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "fluxcache-aof-*.aof")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}
