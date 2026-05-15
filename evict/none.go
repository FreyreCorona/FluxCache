package evict

import "time"

type NoEviction struct{}

func NewNoEviction() *NoEviction { return &NoEviction{} }

func (p *NoEviction) Name() string { return "none" }

func (p *NoEviction) Record(string) {}

func (p *NoEviction) Delete(string) {}

func (p *NoEviction) Evict([]string, map[string]time.Time) string { return "" }
