package persistence_test

import (
	"testing"

	"github.com/FreyreCorona/FluxCache/persistence"
)

func TestNullPersistence(t *testing.T) {
	n := persistence.NewNullPersistence()

	cmd := persistence.Command{Name: "SET", Args: []string{"k", "v"}}
	if err := n.Write(cmd); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	count := 0
	if err := n.Replay(func(cmd persistence.Command) {
		count++
	}); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 replayed commands, got %d", count)
	}

	if err := n.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
