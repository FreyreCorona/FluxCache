package persistence

type NullPersistence struct{}

func NewNullPersistence() *NullPersistence {
	return &NullPersistence{}
}

func (n *NullPersistence) Write(Command) error { return nil }

func (n *NullPersistence) Replay(func(Command)) error { return nil }

func (n *NullPersistence) Close() error { return nil }
