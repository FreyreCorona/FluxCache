package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CommandsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fluxcache_commands_total",
		Help: "Total number of processed commands.",
	}, []string{"command"})

	CommandDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "fluxcache_command_duration_seconds",
		Help:    "Command latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"command"})

	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fluxcache_active_connections",
		Help: "Current number of active client connections.",
	})

	MemoryBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fluxcache_memory_bytes",
		Help: "Current Go runtime memory usage in bytes (Alloc).",
	})

	KeyCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fluxcache_keys_total",
		Help: "Total number of keys in the store.",
	})

	EvictionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fluxcache_evictions_total",
		Help: "Total number of evicted keys.",
	})

	ExpirationsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fluxcache_expirations_total",
		Help: "Total number of expired keys removed.",
	})
)
