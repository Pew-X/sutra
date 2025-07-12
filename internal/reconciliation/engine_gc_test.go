package reconciliation

import (
	"testing"
	"time"

	"github.com/Pew-X/sutra/internal/core"
)

func TestEngineRemoveExpiredKpaks(t *testing.T) {
	engine := NewEngine()

	t.Run("RemoveExpiredKpaks removes only expired k-paks", func(t *testing.T) {
		// Add non-expiring k-pak
		kpak1 := core.NewKpakWithTTL("subject1", "predicate1", "value1", "source1", 1.0, 0)
		engine.Reconcile(kpak1)

		// Add k-pak that expires in the future
		kpak2 := core.NewKpakWithTTL("subject2", "predicate2", "value2", "source2", 1.0, 3600)
		engine.Reconcile(kpak2)

		// Add expired k-pak
		kpak3 := core.NewKpakWithTTL("subject3", "predicate3", "value3", "source3", 1.0, 60)
		kpak3.ExpiresAt = time.Now().Unix() - 3600 // 1 hour ago
		engine.Reconcile(kpak3)

		// Add another expired k-pak
		kpak4 := core.NewKpakWithTTL("subject3", "predicate4", "value4", "source4", 1.0, 60)
		kpak4.ExpiresAt = time.Now().Unix() - 1800 // 30 minutes ago
		engine.Reconcile(kpak4)

		// Remove expired k-paks
		removedCount := engine.RemoveExpiredKpaks()

		// Should have removed 2 expired k-paks
		if removedCount != 2 {
			t.Errorf("Expected to remove 2 expired k-paks, removed %d", removedCount)
		}

		// Check remaining k-paks
		allTruths := engine.GetAllTruths()
		if len(allTruths) != 2 {
			t.Errorf("Expected 2 remaining k-paks, got %d", len(allTruths))
		}

		// Verify the correct k-paks remain
		remainingSubjects := make(map[string]bool)
		for _, kpak := range allTruths {
			remainingSubjects[kpak.Subject] = true
		}

		if !remainingSubjects["subject1"] || !remainingSubjects["subject2"] {
			t.Error("Wrong k-paks were removed")
		}

		if remainingSubjects["subject3"] {
			t.Error("Expired k-paks from subject3 should have been removed")
		}
	})

	t.Run("RemoveExpiredKpaks updates subject index correctly", func(t *testing.T) {
		engine := NewEngine()

		// Add multiple k-paks for the same subject
		kpak1 := core.NewKpakWithTTL("subject1", "predicate1", "value1", "source1", 1.0, 0)
		engine.Reconcile(kpak1)

		kpak2 := core.NewKpakWithTTL("subject1", "predicate2", "value2", "source2", 1.0, 60)
		kpak2.ExpiresAt = time.Now().Unix() - 3600 // expired
		engine.Reconcile(kpak2)

		// Query by subject should return both initially
		results := engine.QueryBySubject("subject1")
		if len(results) != 2 {
			t.Errorf("Expected 2 k-paks for subject1, got %d", len(results))
		}

		// Remove expired k-paks
		removedCount := engine.RemoveExpiredKpaks()
		if removedCount != 1 {
			t.Errorf("Expected to remove 1 expired k-pak, removed %d", removedCount)
		}

		// Query by subject should return only the non-expired one
		results = engine.QueryBySubject("subject1")
		if len(results) != 1 {
			t.Errorf("Expected 1 k-pak for subject1 after GC, got %d", len(results))
		}

		if results[0].Predicate != "predicate1" {
			t.Errorf("Wrong k-pak remained, expected predicate1, got %s", results[0].Predicate)
		}
	})

	t.Run("RemoveExpiredKpaks removes subject from index when all k-paks expire", func(t *testing.T) {
		engine := NewEngine()

		// Add k-paks for a subject that will all expire
		kpak1 := core.NewKpakWithTTL("temp_subject", "predicate1", "value1", "source1", 1.0, 60)
		kpak1.ExpiresAt = time.Now().Unix() - 3600 // expired
		engine.Reconcile(kpak1)

		kpak2 := core.NewKpakWithTTL("temp_subject", "predicate2", "value2", "source2", 1.0, 60)
		kpak2.ExpiresAt = time.Now().Unix() - 1800 // expired
		engine.Reconcile(kpak2)

		// Verify k-paks exist
		results := engine.QueryBySubject("temp_subject")
		if len(results) != 2 {
			t.Errorf("Expected 2 k-paks for temp_subject, got %d", len(results))
		}

		// Remove expired k-paks
		removedCount := engine.RemoveExpiredKpaks()
		if removedCount != 2 {
			t.Errorf("Expected to remove 2 expired k-paks, removed %d", removedCount)
		}

		// Query should return empty results
		results = engine.QueryBySubject("temp_subject")
		if len(results) != 0 {
			t.Errorf("Expected 0 k-paks for temp_subject after GC, got %d", len(results))
		}

		// Check engine stats to ensure subject index was cleaned up
		stats := engine.GetStats()
		totalSubjects := stats["total_subjects"].(int)
		if totalSubjects != 0 {
			t.Errorf("Expected 0 subjects after all k-paks expired, got %d", totalSubjects)
		}
	})

	t.Run("RemoveExpiredKpaks with no expired k-paks returns 0", func(t *testing.T) {
		engine := NewEngine()

		// Add only non-expiring k-paks
		kpak1 := core.NewKpakWithTTL("subject1", "predicate1", "value1", "source1", 1.0, 0)
		engine.Reconcile(kpak1)

		kpak2 := core.NewKpakWithTTL("subject2", "predicate2", "value2", "source2", 1.0, 3600)
		engine.Reconcile(kpak2)

		// Remove expired k-paks
		removedCount := engine.RemoveExpiredKpaks()

		// Should remove nothing
		if removedCount != 0 {
			t.Errorf("Expected to remove 0 k-paks, removed %d", removedCount)
		}

		// All k-paks should remain
		allTruths := engine.GetAllTruths()
		if len(allTruths) != 2 {
			t.Errorf("Expected 2 remaining k-paks, got %d", len(allTruths))
		}
	})
}
