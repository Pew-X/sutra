package monitoring

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()
	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}

	if metrics.ingestRateTracker == nil {
		t.Fatal("Ingest rate tracker not initialized")
	}

	if metrics.queryRateTracker == nil {
		t.Fatal("Query rate tracker not initialized")
	}

	if metrics.activeSources == nil {
		t.Fatal("Active sources not initialized")
	}

	// Verify start time is recent
	now := time.Now()
	if now.Sub(metrics.startTime) > time.Second {
		t.Fatal("Start time should be recent")
	}
}

func TestNewRateTracker(t *testing.T) {
	window := time.Minute
	tracker := NewRateTracker(window)

	if tracker == nil {
		t.Fatal("NewRateTracker returned nil")
	}

	if tracker.maxAge != window {
		t.Fatalf("Expected max age %v, got %v", window, tracker.maxAge)
	}

	if tracker.events == nil {
		t.Fatal("Events slice not initialized")
	}

	if len(tracker.events) != 0 {
		t.Fatal("Events slice should be empty initially")
	}
}

func TestRateTracker_Record(t *testing.T) {
	tracker := NewRateTracker(time.Minute)

	// Initially no events
	rate := tracker.GetRate()
	if rate != 0 {
		t.Fatalf("Expected rate 0, got %d", rate)
	}

	// Record one event
	tracker.Record()
	rate = tracker.GetRate()
	if rate != 1 {
		t.Fatalf("Expected rate 1, got %d", rate)
	}

	// Record more events
	tracker.Record()
	tracker.Record()
	rate = tracker.GetRate()
	if rate != 3 {
		t.Fatalf("Expected rate 3, got %d", rate)
	}
}

func TestRateTracker_CleanOldEvents(t *testing.T) {
	// Use very short window for testing
	tracker := NewRateTracker(100 * time.Millisecond)

	// Record some events
	tracker.Record()
	tracker.Record()

	rate := tracker.GetRate()
	if rate != 2 {
		t.Fatalf("Expected rate 2, got %d", rate)
	}

	// Wait for events to expire
	time.Sleep(150 * time.Millisecond)

	rate = tracker.GetRate()
	if rate != 0 {
		t.Fatalf("Expected rate 0 after expiration, got %d", rate)
	}
}

func TestMetrics_RecordIngest(t *testing.T) {
	metrics := NewMetrics()

	// Initially no ingests
	snapshot := metrics.GetMetrics(0, 0)
	if snapshot.TotalIngested != 0 {
		t.Fatalf("Expected 0 total ingested, got %d", snapshot.TotalIngested)
	}
	if snapshot.TotalAccepted != 0 {
		t.Fatalf("Expected 0 total accepted, got %d", snapshot.TotalAccepted)
	}
	if snapshot.TotalRejected != 0 {
		t.Fatalf("Expected 0 total rejected, got %d", snapshot.TotalRejected)
	}

	// Record accepted ingest
	metrics.RecordIngest("source1", true)
	snapshot = metrics.GetMetrics(0, 0)
	if snapshot.TotalIngested != 1 {
		t.Fatalf("Expected 1 total ingested, got %d", snapshot.TotalIngested)
	}
	if snapshot.TotalAccepted != 1 {
		t.Fatalf("Expected 1 total accepted, got %d", snapshot.TotalAccepted)
	}
	if snapshot.TotalRejected != 0 {
		t.Fatalf("Expected 0 total rejected, got %d", snapshot.TotalRejected)
	}

	// Record rejected ingest
	metrics.RecordIngest("source2", false)
	snapshot = metrics.GetMetrics(0, 0)
	if snapshot.TotalIngested != 2 {
		t.Fatalf("Expected 2 total ingested, got %d", snapshot.TotalIngested)
	}
	if snapshot.TotalAccepted != 1 {
		t.Fatalf("Expected 1 total accepted, got %d", snapshot.TotalAccepted)
	}
	if snapshot.TotalRejected != 1 {
		t.Fatalf("Expected 1 total rejected, got %d", snapshot.TotalRejected)
	}

	// Check ingest rate
	if snapshot.IngestRatePerMin != 2 {
		t.Fatalf("Expected ingest rate 2, got %d", snapshot.IngestRatePerMin)
	}
}

func TestMetrics_RecordQuery(t *testing.T) {
	metrics := NewMetrics()

	// Initially no queries
	snapshot := metrics.GetMetrics(0, 0)
	if snapshot.TotalQueries != 0 {
		t.Fatalf("Expected 0 total queries, got %d", snapshot.TotalQueries)
	}

	// Record queries
	metrics.RecordQuery()
	metrics.RecordQuery()
	metrics.RecordQuery()

	snapshot = metrics.GetMetrics(0, 0)
	if snapshot.TotalQueries != 3 {
		t.Fatalf("Expected 3 total queries, got %d", snapshot.TotalQueries)
	}

	if snapshot.QueryRatePerMin != 3 {
		t.Fatalf("Expected query rate 3, got %d", snapshot.QueryRatePerMin)
	}
}

func TestMetrics_ActiveSources(t *testing.T) {
	metrics := NewMetrics()

	// Initially no active sources
	snapshot := metrics.GetMetrics(0, 0)
	if len(snapshot.ActiveSources) != 0 {
		t.Fatalf("Expected 0 active sources, got %d", len(snapshot.ActiveSources))
	}

	// Record ingests from different sources
	metrics.RecordIngest("source1", true)
	metrics.RecordIngest("source2", false)
	metrics.RecordIngest("source1", true) // Duplicate source

	snapshot = metrics.GetMetrics(0, 0)
	if len(snapshot.ActiveSources) != 2 {
		t.Fatalf("Expected 2 active sources, got %d", len(snapshot.ActiveSources))
	}

	// Check source names
	sources := make(map[string]bool)
	for _, source := range snapshot.ActiveSources {
		sources[source] = true
	}

	if !sources["source1"] || !sources["source2"] {
		t.Fatal("Missing expected active sources")
	}
}

func TestMetrics_GetMetrics(t *testing.T) {
	metrics := NewMetrics()

	// Record some activity
	metrics.RecordIngest("source1", true)
	metrics.RecordIngest("source2", false)
	metrics.RecordQuery()

	// Wait long enough to ensure uptime is recorded
	time.Sleep(1100 * time.Millisecond) // Wait more than 1 second

	snapshot := metrics.GetMetrics(10, 5)

	// Check passed parameters
	if snapshot.TotalKpaks != 10 {
		t.Fatalf("Expected total kpaks 10, got %d", snapshot.TotalKpaks)
	}

	if snapshot.TotalSubjects != 5 {
		t.Fatalf("Expected total subjects 5, got %d", snapshot.TotalSubjects)
	}

	// Check computed values
	if snapshot.TotalIngested != 2 {
		t.Fatalf("Expected total ingested 2, got %d", snapshot.TotalIngested)
	}

	if snapshot.TotalAccepted != 1 {
		t.Fatalf("Expected total accepted 1, got %d", snapshot.TotalAccepted)
	}

	if snapshot.TotalRejected != 1 {
		t.Fatalf("Expected total rejected 1, got %d", snapshot.TotalRejected)
	}

	if snapshot.TotalQueries != 1 {
		t.Fatalf("Expected total queries 1, got %d", snapshot.TotalQueries)
	}

	// Check version
	if snapshot.Version != "1.0.0" {
		t.Fatalf("Expected version '1.0.0', got '%s'", snapshot.Version)
	}

	// Check uptime (should be positive)
	if snapshot.UptimeSeconds <= 0 {
		t.Fatalf("Expected positive uptime, got %d", snapshot.UptimeSeconds)
	}

	// Check memory usage (should be positive)
	if snapshot.MemoryUsageBytes <= 0 {
		t.Fatalf("Expected positive memory usage, got %d", snapshot.MemoryUsageBytes)
	}

	// Check active sources
	if len(snapshot.ActiveSources) != 2 {
		t.Fatalf("Expected 2 active sources, got %d", len(snapshot.ActiveSources))
	}
}

func TestMetrics_GetHealthStatus(t *testing.T) {
	metrics := NewMetrics()

	// Wait long enough to avoid "recently started" status
	time.Sleep(1100 * time.Millisecond)

	// Test status - could be degraded if uptime < 30s or healthy
	status := metrics.GetHealthStatus(100)

	// Accept either healthy or degraded status in tests
	if status.Status != "healthy" && status.Status != "degraded" {
		t.Fatalf("Expected status 'healthy' or 'degraded', got '%s'", status.Status)
	}

	if status.KpakCount != 100 {
		t.Fatalf("Expected kpak count 100, got %d", status.KpakCount)
	}

	if status.UptimeSeconds <= 0 {
		t.Fatalf("Expected positive uptime, got %d", status.UptimeSeconds)
	}

	if status.LastActivity <= 0 {
		t.Fatalf("Expected positive last activity, got %d", status.LastActivity)
	}

	if status.Message == "" {
		t.Fatal("Expected non-empty message")
	}
}

func TestMetrics_GetHealthStatus_RecentStart(t *testing.T) {
	// This test checks if recently started agents are marked as degraded
	// Create metrics instance
	metrics := NewMetrics()

	// Immediately check health status
	status := metrics.GetHealthStatus(0)

	// Should be degraded due to recent start
	if status.Status != "degraded" {
		t.Fatalf("Expected status 'degraded' for recent start, got '%s'", status.Status)
	}

	if !containsString(status.Message, "recently started") {
		t.Fatalf("Expected message about recent start, got '%s'", status.Message)
	}
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewMetrics()

	// Test concurrent access
	const numGoroutines = 10
	const numOperationsPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperationsPerGoroutine; j++ {
				metrics.RecordIngest(fmt.Sprintf("source%d", id), j%2 == 0)
				metrics.RecordQuery()
				metrics.GetMetrics(0, 0)
				metrics.GetHealthStatus(0)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final state
	snapshot := metrics.GetMetrics(0, 0)
	expectedIngested := int64(numGoroutines * numOperationsPerGoroutine)
	expectedQueries := int64(numGoroutines * numOperationsPerGoroutine)

	if snapshot.TotalIngested != expectedIngested {
		t.Fatalf("Expected %d total ingested, got %d", expectedIngested, snapshot.TotalIngested)
	}

	if snapshot.TotalQueries != expectedQueries {
		t.Fatalf("Expected %d total queries, got %d", expectedQueries, snapshot.TotalQueries)
	}

	// Should have multiple active sources
	if len(snapshot.ActiveSources) == 0 {
		t.Fatal("Expected some active sources")
	}
}

func TestMetrics_ActiveSourcesExpiration(t *testing.T) {
	// This test verifies that old sources are removed from active sources
	// We can't easily test this without mocking time, so we'll just verify
	// the basic functionality works
	metrics := NewMetrics()

	metrics.RecordIngest("source1", true)
	snapshot := metrics.GetMetrics(0, 0)

	if len(snapshot.ActiveSources) != 1 {
		t.Fatalf("Expected 1 active source, got %d", len(snapshot.ActiveSources))
	}

	if snapshot.ActiveSources[0] != "source1" {
		t.Fatalf("Expected source1, got %s", snapshot.ActiveSources[0])
	}
}

func TestRateTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewRateTracker(time.Minute)

	// Test concurrent recording
	const numGoroutines = 10
	const numRecordsPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numRecordsPerGoroutine; j++ {
				tracker.Record()
				tracker.GetRate()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final rate
	rate := tracker.GetRate()
	expected := int64(numGoroutines * numRecordsPerGoroutine)
	if rate != expected {
		t.Fatalf("Expected rate %d, got %d", expected, rate)
	}
}

func TestMetricsSnapshot_Structure(t *testing.T) {
	// Test that MetricsSnapshot has all expected fields
	snapshot := MetricsSnapshot{
		TotalKpaks:       10,
		TotalSubjects:    5,
		IngestRatePerMin: 100,
		QueryRatePerMin:  50,
		UptimeSeconds:    3600,
		MemoryUsageBytes: 1024 * 1024,
		CPUUsagePercent:  25.5,
		Version:          "1.0.0",
		ActiveSources:    []string{"source1", "source2"},
		TotalIngested:    200,
		TotalAccepted:    150,
		TotalRejected:    50,
		TotalQueries:     75,
	}

	// Verify all fields are accessible
	if snapshot.TotalKpaks != 10 {
		t.Fatal("TotalKpaks field issue")
	}
	if len(snapshot.ActiveSources) != 2 {
		t.Fatal("ActiveSources field issue")
	}
	if snapshot.CPUUsagePercent != 25.5 {
		t.Fatal("CPUUsagePercent field issue")
	}
}

func TestHealthStatus_Structure(t *testing.T) {
	// Test that HealthStatus has all expected fields
	status := HealthStatus{
		Status:        "healthy",
		Message:       "All good",
		KpakCount:     100,
		UptimeSeconds: 3600,
		LastActivity:  1640995200,
	}

	// Verify all fields are accessible
	if status.Status != "healthy" {
		t.Fatal("Status field issue")
	}
	if status.Message != "All good" {
		t.Fatal("Message field issue")
	}
	if status.KpakCount != 100 {
		t.Fatal("KpakCount field issue")
	}
	if status.UptimeSeconds != 3600 {
		t.Fatal("UptimeSeconds field issue")
	}
	if status.LastActivity != 1640995200 {
		t.Fatal("LastActivity field issue")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
