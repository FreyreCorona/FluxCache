package persistence

import (
	"encoding/binary"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type WAL struct {
	file      *os.File
	mu        sync.Mutex
	buf       []byte
	batchSize int
	interval  time.Duration
	done      chan struct{}
}

func NewWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	w := &WAL{
		file:      f,
		batchSize: 1000,
		interval:  time.Second,
		done:      make(chan struct{}),
	}
	go w.flushLoop()
	return w, nil
}

func marshalCommand(cmd Command) []byte {
	b := make([]byte, 2+len(cmd.Name)+2)
	binary.BigEndian.PutUint16(b[0:2], uint16(len(cmd.Name)))
	copy(b[2:], cmd.Name)
	binary.BigEndian.PutUint16(b[2+len(cmd.Name):], uint16(len(cmd.Args)))
	for _, arg := range cmd.Args {
		b = append(b, 0, 0, 0, 0)
		binary.BigEndian.PutUint32(b[len(b)-4:], uint32(len(arg)))
		b = append(b, arg...)
	}
	return b
}

func unmarshalCommand(data []byte) (Command, []byte) {
	if len(data) < 4 {
		return Command{}, data
	}
	nameLen := binary.BigEndian.Uint16(data[0:2])
	off := 2
	if int(nameLen) > len(data)-off {
		return Command{}, data
	}
	name := string(data[off : off+int(nameLen)])
	off += int(nameLen)
	if off+2 > len(data) {
		return Command{}, data
	}
	argCount := binary.BigEndian.Uint16(data[off:])
	off += 2
	args := make([]string, 0, argCount)
	for i := uint16(0); i < argCount; i++ {
		if off+4 > len(data) {
			return Command{}, data
		}
		argLen := binary.BigEndian.Uint32(data[off:])
		off += 4
		if off+int(argLen) > len(data) {
			return Command{}, data
		}
		args = append(args, string(data[off:off+int(argLen)]))
		off += int(argLen)
	}
	return Command{Name: strings.ToUpper(name), Args: args}, data[off:]
}

func (w *WAL) Write(cmd Command) error {
	data := marshalCommand(cmd)
	w.mu.Lock()
	w.buf = append(w.buf, data...)
	shouldFlush := len(w.buf) >= w.batchSize
	w.mu.Unlock()

	if shouldFlush {
		return w.Flush()
	}
	return nil
}

func (w *WAL) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) == 0 {
		return nil
	}
	_, err := w.file.Write(w.buf)
	w.buf = w.buf[:0]
	return err
}

func (w *WAL) flushLoop() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.Flush()
		case <-w.done:
			w.Flush()
			return
		}
	}
}

func (w *WAL) Replay(fn func(Command)) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buf) > 0 {
		if _, err := w.file.Write(w.buf); err != nil {
			return err
		}
		w.buf = w.buf[:0]
	}

	w.file.Seek(0, 0)
	data, err := io.ReadAll(w.file)
	if err != nil {
		return err
	}

	for len(data) > 0 {
		var cmd Command
		cmd, data = unmarshalCommand(data)
		if cmd.Name != "" {
			fn(cmd)
		}
	}

	_, err = w.file.Seek(0, 2)
	return err
}

func (w *WAL) Close() error {
	close(w.done)
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) > 0 {
		w.file.Write(w.buf)
		w.buf = w.buf[:0]
	}
	return w.file.Close()
}
