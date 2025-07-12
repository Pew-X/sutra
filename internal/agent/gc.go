// Garbage collection for expired k-paks

package agent

import (
	"log"
	"sync"
	"time"

	"github.com/Pew-X/sutra/internal/reconciliation"
)

// GarbageCollector manages automatic cleanup of expired k-paks.
type GarbageCollector struct {
	engine          *reconciliation.Engine
	intervalSeconds int64
	enabled         bool
	ticker          *time.Ticker
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mutex           sync.Mutex
	running         bool
}

// NewGarbageCollector creates a new garbage collector.
func NewGarbageCollector(engine *reconciliation.Engine, intervalSeconds int64, enabled bool) *GarbageCollector {
	if intervalSeconds <= 0 {
		intervalSeconds = 300 // Default to 5 minutes
	}

	return &GarbageCollector{
		engine:          engine,
		intervalSeconds: intervalSeconds,
		enabled:         enabled,
		stopChan:        make(chan struct{}),
	}
}

// Start begins the garbage collection process.
func (gc *GarbageCollector) Start() {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if !gc.enabled || gc.running {
		return
	}

	gc.running = true
	gc.ticker = time.NewTicker(time.Duration(gc.intervalSeconds) * time.Second)

	gc.wg.Add(1)
	go gc.run()

	log.Printf("Garbage collector started with interval %d seconds", gc.intervalSeconds)
}

// Stop gracefully stops the garbage collection process.
func (gc *GarbageCollector) Stop() {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	if !gc.running {
		return
	}

	gc.running = false
	close(gc.stopChan)

	if gc.ticker != nil {
		gc.ticker.Stop()
	}

	gc.wg.Wait()
	log.Println("Garbage collector stopped")
}

// run is the main garbage collection loop.
func (gc *GarbageCollector) run() {
	defer gc.wg.Done()

	for {
		select {
		case <-gc.ticker.C:
			gc.collectGarbage()
		case <-gc.stopChan:
			return
		}
	}
}

// collectGarbage removes expired k-paks from the engine.
func (gc *GarbageCollector) collectGarbage() {
	start := time.Now()
	removed := gc.engine.RemoveExpiredKpaks()
	duration := time.Since(start)

	if removed > 0 {
		log.Printf("Garbage collection completed: removed %d expired k-paks in %v", removed, duration)
	}
}

// GetStats returns garbage collection statistics.
func (gc *GarbageCollector) GetStats() map[string]interface{} {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	return map[string]interface{}{
		"enabled":          gc.enabled,
		"running":          gc.running,
		"interval_seconds": gc.intervalSeconds,
	}
}
