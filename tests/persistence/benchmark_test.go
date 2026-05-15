package persistence_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/FreyreCorona/FluxCache/persistence"
)

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
