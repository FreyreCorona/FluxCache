package persistence_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/FreyreCorona/FluxCache/persistence"
)

func TestAOFWriteReplay(t *testing.T) {
	aof, cleanup := tempAOF(t)
	testPersistence(t, aof, cleanup)
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
	f, err := os.CreateTemp("", "fluxcache-aof-*.aof")
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

func BenchmarkAOFWrite(b *testing.B) {
	f, err := os.CreateTemp("", "fluxcache-aof-bench-*.aof")
	if err != nil {
		b.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	aof, err := persistence.NewAOF(f.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer aof.Close()

	cmds := make([]persistence.Command, b.N)
	for i := range cmds {
		cmds[i] = persistence.Command{
			Name: "SET",
			Args: []string{fmt.Sprintf("key-%d", i), fmt.Sprintf("val-%d", i)},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aof.Write(cmds[i])
	}
}
