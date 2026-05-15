package persistence

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/FreyreCorona/FluxCache/resp"
)

// AOF implements append-only file persistence using the RESP protocol.
type AOF struct {
	file *os.File
	mu   sync.Mutex
}

// NewAOF opens or creates an AOF file at the given path.
func NewAOF(path string) (*AOF, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	a := &AOF{file: f}

	go func() {
		for {
			a.mu.Lock()
			a.file.Sync()
			a.mu.Unlock()
			time.Sleep(time.Second)
		}
	}()

	return a, nil
}

// Write appends a marshalled command to the AOF file.
func (a *AOF) Write(cmd Command) error {
	v := resp.Value{Type: resp.TypeArray, Array: make([]resp.Value, 0, len(cmd.Args)+1)}
	v.Array = append(v.Array, resp.Value{Type: resp.TypeBulk, Bulk: cmd.Name})
	for _, arg := range cmd.Args {
		v.Array = append(v.Array, resp.Value{Type: resp.TypeBulk, Bulk: arg})
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.file.Write(v.Marshal())
	return err
}

// Replay reads all commands from the AOF file and calls fn for each.
func (a *AOF) Replay(fn func(Command)) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.file.Seek(0, 0)

	r := resp.NewResp(a.file)
	for {
		v, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if v.Type != resp.TypeArray || len(v.Array) == 0 {
			continue
		}

		cmd := Command{
			Name: strings.ToUpper(v.Array[0].Bulk),
			Args: make([]string, 0, len(v.Array)-1),
		}
		for _, arg := range v.Array[1:] {
			cmd.Args = append(cmd.Args, arg.Bulk)
		}
		fn(cmd)
	}

	_, err := a.file.Seek(0, 2)
	return err
}

// Close closes the underlying AOF file.
func (a *AOF) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.file.Close()
}
