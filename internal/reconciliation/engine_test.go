package reconciliation

import (
	"fmt"
	"testing"
	"time"

	"github.com/Pew-X/sutra/internal/core"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("NewEngine returned nil")
	}
	if engine.truthStore == nil {
		t.Fatal("truthStore is nil")
	}
	if engine.subjectIndex == nil {
		t.Fatal("subjectIndex is nil")
	}
	if len(engine.truthStore) != 0 {
		t.Fatal("truthStore should be empty initially")
	}
	if len(engine.subjectIndex) != 0 {
		t.Fatal("subjectIndex should be empty initially")
	}
}

func TestReconcile_NewKpak(t *testing.T) {
	engine := NewEngine()

	kpak := core.NewKpak("Alice", "age", "25", "TestSource", 1.0)

	// First time we see this k-pak should be accepted
	accepted := engine.Reconcile(kpak)
	if !accepted {
		t.Fatal("New k-pak should be accepted")
	}

	// Verify it's stored
	stored := engine.truthStore[kpak.SPID]
	if stored == nil {
		t.Fatal("K-pak should be stored after reconciliation")
	}
	if stored.Subject != "Alice" || stored.Predicate != "age" || stored.Object != "25" {
		t.Fatal("Stored k-pak has incorrect data")
	}

	// Verify subject index is updated
	spids, exists := engine.subjectIndex["Alice"]
	if !exists {
		t.Fatal("Subject index should contain Alice")
	}
	if _, exists := spids[kpak.SPID]; !exists {
		t.Fatal("Subject index should contain k-pak SPID")
	}
}

func TestReconcile_Conflict_HigherTrust(t *testing.T) {
	engine := NewEngine()

	// Add initial k-pak with lower trust
	kpak1 := core.NewKpak("Alice", "age", "25", "TestSource1", 0.5)
	accepted := engine.Reconcile(kpak1)
	if !accepted {
		t.Fatal("First k-pak should be accepted")
	}

	// Add conflicting k-pak with higher trust
	kpak2 := core.NewKpak("Alice", "age", "26", "TestSource2", 0.8)
	accepted = engine.Reconcile(kpak2)
	if !accepted {
		t.Fatal("Higher trust k-pak should be accepted")
	}

	// Verify the higher trust k-pak is stored
	stored := engine.truthStore[kpak1.SPID] // SPID should be same (same subject+predicate)
	if stored.Object != "26" {
		t.Fatalf("Expected object '26', got '%s'", stored.Object)
	}
	if stored.Confidence != 0.8 {
		t.Fatalf("Expected confidence 0.8, got %f", stored.Confidence)
	}
}

func TestReconcile_Conflict_LowerTrust(t *testing.T) {
	engine := NewEngine()

	// Add initial k-pak with higher trust
	kpak1 := core.NewKpak("Alice", "age", "25", "TestSource1", 0.8)
	accepted := engine.Reconcile(kpak1)
	if !accepted {
		t.Fatal("First k-pak should be accepted")
	}

	// Add conflicting k-pak with lower trust
	kpak2 := core.NewKpak("Alice", "age", "26", "TestSource2", 0.5)
	accepted = engine.Reconcile(kpak2)
	if accepted {
		t.Fatal("Lower trust k-pak should be rejected")
	}

	// Verify original k-pak is still stored
	stored := engine.truthStore[kpak1.SPID]
	if stored.Object != "25" {
		t.Fatalf("Expected object '25', got '%s'", stored.Object)
	}
	if stored.Confidence != 0.8 {
		t.Fatalf("Expected confidence 0.8, got %f", stored.Confidence)
	}
}

func TestReconcile_SameKpak(t *testing.T) {
	engine := NewEngine()

	kpak := core.NewKpak("Alice", "age", "25", "TestSource", 0.8)

	// Accept first time
	accepted := engine.Reconcile(kpak)
	if !accepted {
		t.Fatal("First k-pak should be accepted")
	}

	// Same k-pak again should be rejected (not new truth)
	accepted = engine.Reconcile(kpak)
	if accepted {
		t.Fatal("Identical k-pak should be rejected")
	}
}

func TestQueryBySubject(t *testing.T) {
	engine := NewEngine()

	// Add multiple k-paks for same subject
	kpak1 := core.NewKpak("Alice", "age", "25", "TestSource", 0.8)
	kpak2 := core.NewKpak("Alice", "height", "5'6\"", "TestSource", 0.7)
	kpak3 := core.NewKpak("Bob", "age", "30", "TestSource", 0.9)

	engine.Reconcile(kpak1)
	engine.Reconcile(kpak2)
	engine.Reconcile(kpak3)

	// Query Alice's k-paks
	aliceKpaks := engine.QueryBySubject("Alice")
	if len(aliceKpaks) != 2 {
		t.Fatalf("Expected 2 k-paks for Alice, got %d", len(aliceKpaks))
	}

	// Verify k-paks contain expected data
	found := make(map[string]bool)
	for _, kpak := range aliceKpaks {
		if kpak.Subject != "Alice" {
			t.Fatalf("Expected subject 'Alice', got '%s'", kpak.Subject)
		}
		found[kpak.Predicate] = true
	}

	if !found["age"] || !found["height"] {
		t.Fatal("Missing expected predicates for Alice")
	}

	// Query non-existent subject
	nonExistent := engine.QueryBySubject("Charlie")
	if nonExistent != nil {
		t.Fatal("Non-existent subject should return nil")
	}
}

func TestQueryBySubjectPredicate(t *testing.T) {
	engine := NewEngine()

	kpak := core.NewKpak("Alice", "age", "25", "TestSource", 0.8)
	engine.Reconcile(kpak)

	// Query existing subject+predicate
	result := engine.QueryBySubjectPredicate("Alice", "age")
	if result == nil {
		t.Fatal("Should find existing k-pak")
	}
	if result.Object != "25" {
		t.Fatalf("Expected object '25', got '%s'", result.Object)
	}

	// Query non-existent subject+predicate
	result = engine.QueryBySubjectPredicate("Alice", "height")
	if result != nil {
		t.Fatal("Non-existent subject+predicate should return nil")
	}

	result = engine.QueryBySubjectPredicate("Bob", "age")
	if result != nil {
		t.Fatal("Non-existent subject should return nil")
	}
}

func TestGetAllTruths(t *testing.T) {
	engine := NewEngine()

	// Initially empty
	truths := engine.GetAllTruths()
	if len(truths) != 0 {
		t.Fatal("Initially should have no truths")
	}

	// Add some k-paks
	kpak1 := core.NewKpak("Alice", "age", "25", "TestSource", 0.8)
	kpak2 := core.NewKpak("Bob", "height", "6'0\"", "TestSource", 0.7)

	engine.Reconcile(kpak1)
	engine.Reconcile(kpak2)

	truths = engine.GetAllTruths()
	if len(truths) != 2 {
		t.Fatalf("Expected 2 truths, got %d", len(truths))
	}

	// Verify all truths are present
	subjects := make(map[string]bool)
	for _, kpak := range truths {
		subjects[kpak.Subject] = true
	}

	if !subjects["Alice"] || !subjects["Bob"] {
		t.Fatal("Missing expected subjects in truths")
	}
}

func TestGetStats(t *testing.T) {
	engine := NewEngine()

	// Initially empty
	stats := engine.GetStats()
	if stats["total_kpaks"] != 0 {
		t.Fatal("Initially should have 0 k-paks")
	}
	if stats["total_subjects"] != 0 {
		t.Fatal("Initially should have 0 subjects")
	}

	// Add k-paks
	kpak1 := core.NewKpak("Alice", "age", "25", "TestSource", 0.8)
	kpak2 := core.NewKpak("Alice", "height", "5'6\"", "TestSource", 0.7)
	kpak3 := core.NewKpak("Bob", "age", "30", "TestSource", 0.9)

	engine.Reconcile(kpak1)
	engine.Reconcile(kpak2)
	engine.Reconcile(kpak3)

	stats = engine.GetStats()
	if stats["total_kpaks"] != 3 {
		t.Fatalf("Expected 3 k-paks, got %v", stats["total_kpaks"])
	}
	if stats["total_subjects"] != 2 {
		t.Fatalf("Expected 2 subjects, got %v", stats["total_subjects"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	engine := NewEngine()

	// Test concurrent reconciliation
	const numGoroutines = 10
	const numKpaksPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numKpaksPerGoroutine; j++ {
				kpak := core.NewKpak(
					fmt.Sprintf("Subject%d", j),
					"property",
					fmt.Sprintf("value%d-%d", id, j),
					fmt.Sprintf("Source%d", id),
					0.5,
				)
				engine.Reconcile(kpak)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final state
	stats := engine.GetStats()
	totalKpaks := stats["total_kpaks"].(int)
	if totalKpaks != numKpaksPerGoroutine {
		t.Fatalf("Expected %d k-paks, got %d", numKpaksPerGoroutine, totalKpaks)
	}

	// Verify we can query without race conditions
	for i := 0; i < 10; i++ {
		go func() {
			engine.QueryBySubject("Subject1")
			engine.GetAllTruths()
			engine.GetStats()
		}()
	}

	time.Sleep(100 * time.Millisecond) // Let queries complete
}

func TestTrustBasedReconciliation(t *testing.T) {
	engine := NewEngine()

	// Test various trust scores
	testCases := []struct {
		trust1   float32
		trust2   float32
		expected string // which value should win
	}{
		{0.1, 0.9, "high_trust"},
		{0.9, 0.1, "low_trust"},
		{0.5, 0.5, "low_trust"},  // Equal trust, first wins
		{0.0, 1.0, "high_trust"}, // Second kpak has higher trust
		{1.0, 0.0, "low_trust"},  // First kpak has higher trust
	}

	for i, tc := range testCases {
		// Create fresh subject for each test
		subject := fmt.Sprintf("TestSubject%d", i)

		kpak1 := core.NewKpak(subject, "value", "low_trust", "Source1", tc.trust1)
		kpak2 := core.NewKpak(subject, "value", "high_trust", "Source2", tc.trust2)

		engine.Reconcile(kpak1)
		engine.Reconcile(kpak2)

		result := engine.QueryBySubjectPredicate(subject, "value")
		if result == nil {
			t.Fatalf("Test case %d: no result found", i)
		}

		if result.Object != tc.expected {
			t.Fatalf("Test case %d: expected '%s', got '%s'", i, tc.expected, result.Object)
		}
	}
}
