package persistence

type Command struct {
	Name string
	Args []string
}

type Persistence interface {
	Write(cmd Command) error
	Replay(func(Command)) error
	Close() error
}
