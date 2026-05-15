package persistence_test

import (
	"testing"

	"github.com/FreyreCorona/FluxCache/persistence"
)

func TestDualPersistence(t *testing.T) {
	primary := persistence.NewNullPersistence()
	secondary := persistence.NewNullPersistence()

	d := persistence.NewDualPersistence(primary, secondary)

	cmd := persistence.Command{Name: "SET", Args: []string{"k", "v"}}
	if err := d.Write(cmd); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	count := 0
	if err := d.Replay(func(cmd persistence.Command) {
		count++
	}); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 replayed commands, got %d", count)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestDualPersistenceWithAOF(t *testing.T) {
	aof, cleanupAOF := tempAOF(t)
	defer cleanupAOF()

	null := persistence.NewNullPersistence()
	d := persistence.NewDualPersistence(aof, null)

	testPersistence(t, d, func() {})
}
