package evict

import "time"

// NoEviction implements a policy that never evicts keys.
type NoEviction struct{}

// NewNoEviction creates a new NoEviction policy.
func NewNoEviction() *NoEviction { return &NoEviction{} }

func (p *NoEviction) Name() string { return "none" }

func (p *NoEviction) Record(string) {}

func (p *NoEviction) Delete(string) {}

func (p *NoEviction) Evict([]string, map[string]time.Time) string { return "" }
