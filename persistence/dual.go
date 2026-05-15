package persistence

type DualPersistence struct {
	primary   Persistence
	secondary Persistence
}

func NewDualPersistence(primary, secondary Persistence) *DualPersistence {
	return &DualPersistence{primary: primary, secondary: secondary}
}

func (d *DualPersistence) Write(cmd Command) error {
	err1 := d.primary.Write(cmd)
	err2 := d.secondary.Write(cmd)
	if err1 != nil {
		return err1
	}
	return err2
}

func (d *DualPersistence) Replay(fn func(Command)) error {
	return d.primary.Replay(fn)
}

func (d *DualPersistence) Close() error {
	err1 := d.primary.Close()
	err2 := d.secondary.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
