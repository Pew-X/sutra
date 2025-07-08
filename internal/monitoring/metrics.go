package monitoring

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks agent performance and health metrics.
type Metrics struct {
	startTime time.Time

	// Atomic counters
	totalIngestedKpaks int64
	totalQueries       int64
	totalAcceptedKpaks int64
	totalRejectedKpaks int64

	// Rate tracking
	ingestRateTracker *RateTracker
	queryRateTracker  *RateTracker

	// Sources tracking
	activeSources map[string]time.Time
	sourcesMutex  sync.RWMutex
}

// RateTracker tracks operations per minute.
type RateTracker struct {
	events []time.Time
	mutex  sync.Mutex
	maxAge time.Duration
}

// NewMetrics creates a new metrics tracker.
func NewMetrics() *Metrics {
	return &Metrics{
		startTime:         time.Now(),
		ingestRateTracker: NewRateTracker(time.Minute),
		queryRateTracker:  NewRateTracker(time.Minute),
		activeSources:     make(map[string]time.Time),
	}
}

// NewRateTracker creates a new rate tracker.
func NewRateTracker(window time.Duration) *RateTracker {
	return &RateTracker{
		events: make([]time.Time, 0),
		maxAge: window,
	}
}

// Record records an event in the rate tracker.
func (rt *RateTracker) Record() {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	now := time.Now()
	rt.events = append(rt.events, now)

	// Clean old events
	rt.cleanOldEvents(now)
}

// GetRate returns the current rate (events per tracking window).
func (rt *RateTracker) GetRate() int64 {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	rt.cleanOldEvents(time.Now())
	return int64(len(rt.events))
}

// cleanOldEvents removes events older than the tracking window.
func (rt *RateTracker) cleanOldEvents(now time.Time) {
	cutoff := now.Add(-rt.maxAge)
	validEvents := rt.events[:0]

	for _, event := range rt.events {
		if event.After(cutoff) {
			validEvents = append(validEvents, event)
		}
	}

	rt.events = validEvents
}

// Metric tracking methods

// RecordIngest records a k-pak ingestion.
func (m *Metrics) RecordIngest(source string, accepted bool) {
	atomic.AddInt64(&m.totalIngestedKpaks, 1)
	m.ingestRateTracker.Record()

	if accepted {
		atomic.AddInt64(&m.totalAcceptedKpaks, 1)
	} else {
		atomic.AddInt64(&m.totalRejectedKpaks, 1)
	}

	// Track active source
	m.sourcesMutex.Lock()
	m.activeSources[source] = time.Now()
	m.sourcesMutex.Unlock()
}

// RecordQuery records a query operation.
func (m *Metrics) RecordQuery() {
	atomic.AddInt64(&m.totalQueries, 1)
	m.queryRateTracker.Record()
}

// GetMetrics returns current metric values.
func (m *Metrics) GetMetrics(totalKpaks, totalSubjects int32) MetricsSnapshot {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get active sources (sources active in last 5 minutes)
	m.sourcesMutex.RLock()
	activeSources := make([]string, 0, len(m.activeSources))
	cutoff := time.Now().Add(-5 * time.Minute)
	for source, lastSeen := range m.activeSources {
		if lastSeen.After(cutoff) {
			activeSources = append(activeSources, source)
		}
	}
	m.sourcesMutex.RUnlock()

	return MetricsSnapshot{
		TotalKpaks:       totalKpaks,
		TotalSubjects:    totalSubjects,
		IngestRatePerMin: m.ingestRateTracker.GetRate(),
		QueryRatePerMin:  m.queryRateTracker.GetRate(),
		UptimeSeconds:    int64(time.Since(m.startTime).Seconds()),
		MemoryUsageBytes: int64(memStats.Alloc),
		CPUUsagePercent:  0.0, // TODO: Implement CPU tracking
		Version:          "1.0.0",
		ActiveSources:    activeSources,
		TotalIngested:    atomic.LoadInt64(&m.totalIngestedKpaks),
		TotalAccepted:    atomic.LoadInt64(&m.totalAcceptedKpaks),
		TotalRejected:    atomic.LoadInt64(&m.totalRejectedKpaks),
		TotalQueries:     atomic.LoadInt64(&m.totalQueries),
	}
}

// MetricsSnapshot represents a point-in-time view of metrics.
type MetricsSnapshot struct {
	TotalKpaks       int32    `json:"total_kpaks"`
	TotalSubjects    int32    `json:"total_subjects"`
	IngestRatePerMin int64    `json:"ingest_rate_per_min"`
	QueryRatePerMin  int64    `json:"query_rate_per_min"`
	UptimeSeconds    int64    `json:"uptime_seconds"`
	MemoryUsageBytes int64    `json:"memory_usage_bytes"`
	CPUUsagePercent  float32  `json:"cpu_usage_percent"`
	Version          string   `json:"version"`
	ActiveSources    []string `json:"active_sources"`
	TotalIngested    int64    `json:"total_ingested"`
	TotalAccepted    int64    `json:"total_accepted"`
	TotalRejected    int64    `json:"total_rejected"`
	TotalQueries     int64    `json:"total_queries"`
}

// HealthStatus represents the health status of the agent.
type HealthStatus struct {
	Status        string `json:"status"`         // "healthy", "degraded", "unhealthy"
	Message       string `json:"message"`        // Human-readable status message
	KpakCount     int32  `json:"kpak_count"`     // Total k-paks stored
	UptimeSeconds int64  `json:"uptime_seconds"` // Agent uptime
	LastActivity  int64  `json:"last_activity"`  // Last activity timestamp
}

// GetHealthStatus returns the current health status.
func (m *Metrics) GetHealthStatus(totalKpaks int32) HealthStatus {
	uptime := int64(time.Since(m.startTime).Seconds())

	// Determine health status
	status := "healthy"
	message := "Agent is operating normally"

	// Check for potential issues
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	if memStats.Alloc > 1024*1024*1024 { // > 1GB
		status = "degraded"
		message = "High memory usage detected"
	}

	if uptime < 30 { // Less than 30 seconds uptime
		status = "degraded"
		message = "Agent recently started"
	}

	return HealthStatus{
		Status:        status,
		Message:       message,
		KpakCount:     totalKpaks,
		UptimeSeconds: uptime,
		LastActivity:  time.Now().Unix(),
	}
}
