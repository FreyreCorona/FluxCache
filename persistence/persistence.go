package persistence

// Command represents a single operation to be persisted.
type Command struct {
	Name string
	Args []string
}

// Persistence defines the interface for write-ahead logging and snapshot persistence.
type Persistence interface {
	Write(cmd Command) error
	Replay(func(Command)) error
	Close() error
}
