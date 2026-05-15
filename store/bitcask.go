package store

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
)

const bitcaskMagic = "FCBT01"

// BitcaskStore is a durable store backed by an append-only log file (Bitcask model).
type BitcaskStore struct {
	mu      sync.RWMutex
	file    *os.File
	path    string
	strings map[string]string
	hashes  map[string]map[string]string
}

// NewBitcaskStore opens or creates the Bitcask data file at the given path.
func NewBitcaskStore(path string) (*BitcaskStore, error) {
	s := &BitcaskStore{
		path:    path,
		strings: make(map[string]string),
		hashes:  make(map[string]map[string]string),
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("bitcask: open: %w", err)
	}
	s.file = f

	if err := s.recover(); err != nil {
		f.Close()
		return nil, fmt.Errorf("bitcask: recover: %w", err)
	}

	return s, nil
}

func (s *BitcaskStore) recover() error {
	fi, err := s.file.Stat()
	if err != nil {
		return err
	}
	if fi.Size() == 0 {
		_, err := s.file.Write([]byte(bitcaskMagic))
		return err
	}

	magic := make([]byte, len(bitcaskMagic))
	if _, err := s.file.ReadAt(magic, 0); err != nil {
		return err
	}
	if string(magic) != bitcaskMagic {
		return fmt.Errorf("bitcask: invalid magic")
	}

	pos := int64(len(bitcaskMagic))
	for {
		var op [1]byte
		if _, err := s.file.ReadAt(op[:], pos); err != nil {
			break
		}
		pos++

		switch op[0] {
		case 'S':
			key := readStringAt(s.file, &pos)
			val := readStringAt(s.file, &pos)
			s.strings[key] = val
		case 'D':
			key := readStringAt(s.file, &pos)
			delete(s.strings, key)
			delete(s.hashes, key)
		case 'H':
			hash := readStringAt(s.file, &pos)
			field := readStringAt(s.file, &pos)
			val := readStringAt(s.file, &pos)
			m, ok := s.hashes[hash]
			if !ok {
				m = make(map[string]string)
				s.hashes[hash] = m
			}
			m[field] = val
		}
	}

	_, err = s.file.Seek(0, 2)
	return err
}

func readStringAt(f *os.File, pos *int64) string {
	var lenBuf [4]byte
	if _, err := f.ReadAt(lenBuf[:], *pos); err != nil {
		return ""
	}
	strLen := binary.BigEndian.Uint32(lenBuf[:])
	*pos += 4
	if strLen == 0 {
		return ""
	}
	buf := make([]byte, strLen)
	if _, err := f.ReadAt(buf, *pos); err != nil {
		return ""
	}
	*pos += int64(strLen)
	return string(buf)
}

func (s *BitcaskStore) writeOp(op byte, parts ...string) error {
	buf := []byte{op}
	for _, p := range parts {
		lenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBytes, uint32(len(p)))
		buf = append(buf, lenBytes...)
		buf = append(buf, []byte(p)...)
	}
	_, err := s.file.Write(buf)
	return err
}

func (s *BitcaskStore) Set(key, value string) {
	s.mu.Lock()
	s.strings[key] = value
	s.writeOp('S', key, value)
	s.mu.Unlock()
}

func (s *BitcaskStore) Get(key string) (string, bool) {
	s.mu.RLock()
	val, ok := s.strings[key]
	s.mu.RUnlock()
	return val, ok
}

func (s *BitcaskStore) Del(key string) {
	s.mu.Lock()
	delete(s.strings, key)
	delete(s.hashes, key)
	s.writeOp('D', key)
	s.mu.Unlock()
}

func (s *BitcaskStore) HSet(hash, key, value string) {
	s.mu.Lock()
	m, ok := s.hashes[hash]
	if !ok {
		m = make(map[string]string)
		s.hashes[hash] = m
	}
	m[key] = value
	s.writeOp('H', hash, key, value)
	s.mu.Unlock()
}

func (s *BitcaskStore) HGet(hash, key string) (string, bool) {
	s.mu.RLock()
	m, ok := s.hashes[hash]
	if !ok {
		s.mu.RUnlock()
		return "", false
	}
	val, ok := m[key]
	s.mu.RUnlock()
	return val, ok
}

func (s *BitcaskStore) HGetAll(hash string) map[string]string {
	s.mu.RLock()
	m, ok := s.hashes[hash]
	if !ok {
		s.mu.RUnlock()
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	s.mu.RUnlock()
	return out
}

func (s *BitcaskStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
