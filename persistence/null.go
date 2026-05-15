package persistence

// NullPersistence is a no-op implementation of Persistence.
type NullPersistence struct{}

// NewNullPersistence returns a no-op persistence backend.
func NewNullPersistence() *NullPersistence {
	return &NullPersistence{}
}

// Write is a no-op that always returns nil.
func (n *NullPersistence) Write(Command) error { return nil }

// Replay is a no-op that always returns nil.
func (n *NullPersistence) Replay(func(Command)) error { return nil }

// Close is a no-op that always returns nil.
func (n *NullPersistence) Close() error { return nil }
