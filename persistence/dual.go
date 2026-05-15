package persistence

// DualPersistence writes to two Persistence backends simultaneously.
type DualPersistence struct {
	primary   Persistence
	secondary Persistence
}

// NewDualPersistence creates a DualPersistence wrapping two backends.
func NewDualPersistence(primary, secondary Persistence) *DualPersistence {
	return &DualPersistence{primary: primary, secondary: secondary}
}

// Write writes the command to both primary and secondary backends.
func (d *DualPersistence) Write(cmd Command) error {
	err1 := d.primary.Write(cmd)
	err2 := d.secondary.Write(cmd)
	if err1 != nil {
		return err1
	}
	return err2
}

// Replay replays commands from the primary backend only.
func (d *DualPersistence) Replay(fn func(Command)) error {
	return d.primary.Replay(fn)
}

// Close closes both the primary and secondary backends.
func (d *DualPersistence) Close() error {
	err1 := d.primary.Close()
	err2 := d.secondary.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
