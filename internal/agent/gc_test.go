package agent

import (
	"sync"
	"testing"
	"time"

	"github.com/Pew-X/sutra/internal/core"
	"github.com/Pew-X/sutra/internal/reconciliation"
)

func TestGarbageCollector(t *testing.T) {
	t.Run("NewGarbageCollector creates GC with correct settings", func(t *testing.T) {
		engine := reconciliation.NewEngine()
		gc := NewGarbageCollector(engine, 60, true)

		if gc.engine != engine {
			t.Error("Engine not set correctly")
		}
		if gc.intervalSeconds != 60 {
			t.Errorf("Expected interval 60, got %d", gc.intervalSeconds)
		}
		if !gc.enabled {
			t.Error("GC should be enabled")
		}
	})

	t.Run("NewGarbageCollector sets default interval for invalid input", func(t *testing.T) {
		engine := reconciliation.NewEngine()
		gc := NewGarbageCollector(engine, 0, true)

		if gc.intervalSeconds != 300 {
			t.Errorf("Expected default interval 300, got %d", gc.intervalSeconds)
		}
	})

	t.Run("GarbageCollector Start and Stop", func(t *testing.T) {
		engine := reconciliation.NewEngine()
		gc := NewGarbageCollector(engine, 1, true) // 1 second interval for fast testing

		// Initially should not be running
		stats := gc.GetStats()
		if stats["running"].(bool) {
			t.Error("GC should not be running initially")
		}

		// Start GC
		gc.Start()

		// Should be running now
		stats = gc.GetStats()
		if !stats["running"].(bool) {
			t.Error("GC should be running after Start()")
		}

		// Stop GC
		gc.Stop()

		// Should not be running anymore
		stats = gc.GetStats()
		if stats["running"].(bool) {
			t.Error("GC should not be running after Stop()")
		}
	})

	t.Run("GarbageCollector doesn't start if disabled", func(t *testing.T) {
		engine := reconciliation.NewEngine()
		gc := NewGarbageCollector(engine, 60, false) // disabled

		gc.Start()

		stats := gc.GetStats()
		if stats["running"].(bool) {
			t.Error("Disabled GC should not start")
		}
	})

	t.Run("GarbageCollector multiple Start calls are safe", func(t *testing.T) {
		engine := reconciliation.NewEngine()
		gc := NewGarbageCollector(engine, 60, true)

		gc.Start()
		gc.Start() // Should not panic or cause issues

		stats := gc.GetStats()
		if !stats["running"].(bool) {
			t.Error("GC should be running")
		}

		gc.Stop()
	})

	t.Run("GarbageCollector multiple Stop calls are safe", func(t *testing.T) {
		engine := reconciliation.NewEngine()
		gc := NewGarbageCollector(engine, 60, true)

		gc.Start()
		gc.Stop()
		gc.Stop() // Should not panic or cause issues

		stats := gc.GetStats()
		if stats["running"].(bool) {
			t.Error("GC should not be running after Stop()")
		}
	})
}

func TestGarbageCollectorIntegration(t *testing.T) {
	t.Run("GarbageCollector actually removes expired k-paks", func(t *testing.T) {
		engine := reconciliation.NewEngine()

		// Add some k-paks
		kpak1 := core.NewKpakWithTTL("subject1", "predicate1", "value1", "source1", 1.0, 0) // never expires
		engine.Reconcile(kpak1)

		kpak2 := core.NewKpakWithTTL("subject2", "predicate2", "value2", "source2", 1.0, 60)
		kpak2.ExpiresAt = time.Now().Unix() - 3600 // expired 1 hour ago
		engine.Reconcile(kpak2)

		// Verify k-paks are there
		allTruths := engine.GetAllTruths()
		if len(allTruths) != 2 {
			t.Errorf("Expected 2 k-paks initially, got %d", len(allTruths))
		}

		// Create GC with very short interval for testing
		gc := NewGarbageCollector(engine, 1, true)

		// Manually trigger garbage collection
		gc.collectGarbage()

		// Should have removed the expired k-pak
		allTruths = engine.GetAllTruths()
		if len(allTruths) != 1 {
			t.Errorf("Expected 1 k-pak after GC, got %d", len(allTruths))
		}

		if allTruths[0].Subject != "subject1" {
			t.Errorf("Wrong k-pak remained, expected subject1, got %s", allTruths[0].Subject)
		}
	})

	t.Run("GarbageCollector runs automatically", func(t *testing.T) {
		engine := reconciliation.NewEngine()

		// Add expired k-pak
		kpak := core.NewKpakWithTTL("subject1", "predicate1", "value1", "source1", 1.0, 60)
		kpak.ExpiresAt = time.Now().Unix() - 3600 // expired
		engine.Reconcile(kpak)

		// Verify k-pak is there
		allTruths := engine.GetAllTruths()
		if len(allTruths) != 1 {
			t.Errorf("Expected 1 k-pak initially, got %d", len(allTruths))
		}

		// Start GC with very short interval
		gc := NewGarbageCollector(engine, 1, true) // 1 second interval
		gc.Start()

		// Wait for at least one GC cycle
		time.Sleep(2 * time.Second)

		// Stop GC
		gc.Stop()

		// Should have removed the expired k-pak
		allTruths = engine.GetAllTruths()
		if len(allTruths) != 0 {
			t.Errorf("Expected 0 k-paks after automatic GC, got %d", len(allTruths))
		}
	})
}

func TestGarbageCollectorConcurrency(t *testing.T) {
	t.Run("GarbageCollector is thread-safe", func(t *testing.T) {
		engine := reconciliation.NewEngine()
		gc := NewGarbageCollector(engine, 60, true)

		var wg sync.WaitGroup
		concurrency := 10

		// Start multiple goroutines trying to start/stop GC
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				gc.Start()
				time.Sleep(10 * time.Millisecond)
				gc.Stop()
			}()
		}

		wg.Wait()

		// Should not be running at the end
		stats := gc.GetStats()
		if stats["running"].(bool) {
			t.Error("GC should not be running after concurrent start/stop")
		}
	})
}
