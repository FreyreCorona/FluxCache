package telemetry

import (
	"runtime"
	"time"

	"github.com/FreyreCorona/FluxCache/log"
)

// Collector periodically reads runtime stats and store metrics.
type Collector struct {
	keyCount func() int
	stop     chan struct{}
}

// NewCollector creates a collector that reads key count from the given func.
func NewCollector(keyCount func() int) *Collector {
	return &Collector{keyCount: keyCount}
}

// Start begins periodic collection on a 10s interval.
func (c *Collector) Start() {
	c.stop = make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.collect()
			case <-c.stop:
				return
			}
		}
	}()
	log.Debug("metrics collector started")
}

// Stop halts periodic collection.
func (c *Collector) Stop() {
	if c.stop != nil {
		close(c.stop)
	}
}

func (c *Collector) collect() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	MemoryBytes.Set(float64(m.Alloc))

	if c.keyCount != nil {
		KeyCount.Set(float64(c.keyCount()))
	}
}
