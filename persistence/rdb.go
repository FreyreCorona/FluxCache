package persistence

import (
	"encoding/binary"
	"os"
	"sync"
	"time"
)

const rdbMagic = "FLXCRDB"
const rdbVersion uint32 = 1

// RDB implements periodic snapshot persistence to a binary file.
type RDB struct {
	mu       sync.Mutex
	path     string
	strings  map[string]string
	hashes   map[string]map[string]string
	interval time.Duration
	done     chan struct{}
}

// NewRDB creates an RDB snapshotter with a default 5-second interval.
func NewRDB(path string) (*RDB, error) {
	r := &RDB{
		path:     path,
		strings:  make(map[string]string),
		hashes:   make(map[string]map[string]string),
		interval: 5 * time.Second,
		done:     make(chan struct{}),
	}
	go r.snapshotLoop()
	return r, nil
}

// NewRDBWithInterval creates an RDB snapshotter with a custom interval.
func NewRDBWithInterval(path string, interval time.Duration) (*RDB, error) {
	r, err := NewRDB(path)
	if err != nil {
		return nil, err
	}
	r.interval = interval
	return r, nil
}

// Write records a command in the in-memory state for the next snapshot.
func (r *RDB) Write(cmd Command) error {
	r.mu.Lock()
	switch cmd.Name {
	case "SET":
		if len(cmd.Args) >= 2 {
			r.strings[cmd.Args[0]] = cmd.Args[1]
		}
	case "HSET":
		if len(cmd.Args) >= 3 {
			h, ok := r.hashes[cmd.Args[0]]
			if !ok {
				h = make(map[string]string)
				r.hashes[cmd.Args[0]] = h
			}
			h[cmd.Args[1]] = cmd.Args[2]
		}
	}
	r.mu.Unlock()
	return nil
}

// Snapshot writes the current in-memory state to disk immediately.
func (r *RDB) Snapshot() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writeSnapshot()
}

func (r *RDB) writeSnapshot() error {
	f, err := os.Create(r.path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write([]byte(rdbMagic))
	version := make([]byte, 4)
	binary.BigEndian.PutUint32(version, rdbVersion)
	f.Write(version)

	strLen := make([]byte, 4)
	binary.BigEndian.PutUint32(strLen, uint32(len(r.strings)))
	f.Write(strLen)
	for k, v := range r.strings {
		writeString(f, k)
		writeString(f, v)
	}

	hashLen := make([]byte, 4)
	binary.BigEndian.PutUint32(hashLen, uint32(len(r.hashes)))
	f.Write(hashLen)
	for hk, fields := range r.hashes {
		writeString(f, hk)
		fieldCount := make([]byte, 4)
		binary.BigEndian.PutUint32(fieldCount, uint32(len(fields)))
		f.Write(fieldCount)
		for fk, fv := range fields {
			writeString(f, fk)
			writeString(f, fv)
		}
	}

	return nil
}

func writeString(f *os.File, s string) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(len(s)))
	f.Write(buf)
	f.WriteString(s)
}

func (r *RDB) snapshotLoop() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.Snapshot()
		case <-r.done:
			r.Snapshot()
			return
		}
	}
}

// Replay reads the snapshot file and replays its commands via fn.
func (r *RDB) Replay(fn func(Command)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := os.Open(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if fi.Size() == 0 {
		return nil
	}

	magic := make([]byte, len(rdbMagic))
	if _, err := f.Read(magic); err != nil {
		return err
	}

	versionBuf := make([]byte, 4)
	if _, err := f.Read(versionBuf); err != nil {
		return err
	}

	strCountBuf := make([]byte, 4)
	if _, err := f.Read(strCountBuf); err != nil {
		return err
	}
	strCount := binary.BigEndian.Uint32(strCountBuf)

	for range strCount {
		k := readString(f)
		v := readString(f)
		if k == "" && v == "" {
			continue
		}
		fn(Command{Name: "SET", Args: []string{k, v}})
	}

	hashCountBuf := make([]byte, 4)
	if _, err := f.Read(hashCountBuf); err != nil {
		return err
	}
	hashCount := binary.BigEndian.Uint32(hashCountBuf)

	for range hashCount {
		hk := readString(f)
		fieldCountBuf := make([]byte, 4)
		if _, err := f.Read(fieldCountBuf); err != nil {
			return err
		}
		fieldCount := binary.BigEndian.Uint32(fieldCountBuf)
		for range fieldCount {
			fk := readString(f)
			fv := readString(f)
			fn(Command{Name: "HSET", Args: []string{hk, fk, fv}})
		}
	}

	return nil
}

func readString(f *os.File) string {
	lenBuf := make([]byte, 4)
	if _, err := f.Read(lenBuf); err != nil {
		return ""
	}
	strLen := binary.BigEndian.Uint32(lenBuf)
	if strLen == 0 {
		return ""
	}
	buf := make([]byte, strLen)
	if _, err := f.Read(buf); err != nil {
		return ""
	}
	return string(buf)
}

// Close stops the snapshot loop.
func (r *RDB) Close() error {
	close(r.done)
	return nil
}
