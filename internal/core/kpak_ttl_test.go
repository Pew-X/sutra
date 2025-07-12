package core

import (
	"testing"
	"time"
)

func TestKpakTTL(t *testing.T) {
	t.Run("NewKpakWithTTL creates k-pak with correct expiry", func(t *testing.T) {
		ttl := int64(60) // 60 seconds
		kpak := NewKpakWithTTL("subject1", "predicate1", "value1", "test-source", 1.0, ttl)

		expectedExpiry := time.Now().Unix() + ttl
		if kpak.ExpiresAt < expectedExpiry-1 || kpak.ExpiresAt > expectedExpiry+1 {
			t.Errorf("Expected ExpiresAt around %d, got %d", expectedExpiry, kpak.ExpiresAt)
		}
	})

	t.Run("NewKpakWithTTL with zero TTL never expires", func(t *testing.T) {
		kpak := NewKpakWithTTL("subject1", "predicate1", "value1", "test-source", 1.0, 0)

		if kpak.ExpiresAt != 0 {
			t.Errorf("Expected ExpiresAt to be 0 for never expires, got %d", kpak.ExpiresAt)
		}
	})

	t.Run("NewKpak defaults to never expires", func(t *testing.T) {
		kpak := NewKpak("subject1", "predicate1", "value1", "test-source", 1.0)

		if kpak.ExpiresAt != 0 {
			t.Errorf("Expected ExpiresAt to be 0 for default NewKpak, got %d", kpak.ExpiresAt)
		}
	})
}

func TestKpakIsExpired(t *testing.T) {
	t.Run("k-pak with zero ExpiresAt never expires", func(t *testing.T) {
		kpak := NewKpakWithTTL("subject1", "predicate1", "value1", "test-source", 1.0, 0)

		if kpak.IsExpired() {
			t.Error("K-pak with ExpiresAt=0 should never expire")
		}
	})

	t.Run("k-pak with future ExpiresAt is not expired", func(t *testing.T) {
		futureTime := time.Now().Unix() + 3600 // 1 hour from now
		kpak := NewKpakWithTTL("subject1", "predicate1", "value1", "test-source", 1.0, 3600)
		kpak.ExpiresAt = futureTime

		if kpak.IsExpired() {
			t.Error("K-pak with future ExpiresAt should not be expired")
		}
	})

	t.Run("k-pak with past ExpiresAt is expired", func(t *testing.T) {
		pastTime := time.Now().Unix() - 3600 // 1 hour ago
		kpak := NewKpakWithTTL("subject1", "predicate1", "value1", "test-source", 1.0, 60)
		kpak.ExpiresAt = pastTime

		if !kpak.IsExpired() {
			t.Error("K-pak with past ExpiresAt should be expired")
		}
	})
}

func TestKpakTimeToExpiry(t *testing.T) {
	t.Run("k-pak with zero ExpiresAt returns 0", func(t *testing.T) {
		kpak := NewKpakWithTTL("subject1", "predicate1", "value1", "test-source", 1.0, 0)

		if kpak.TimeToExpiry() != 0 {
			t.Errorf("Expected TimeToExpiry to be 0 for never expires, got %d", kpak.TimeToExpiry())
		}
	})

	t.Run("expired k-pak returns -1", func(t *testing.T) {
		pastTime := time.Now().Unix() - 3600 // 1 hour ago
		kpak := NewKpakWithTTL("subject1", "predicate1", "value1", "test-source", 1.0, 60)
		kpak.ExpiresAt = pastTime

		if kpak.TimeToExpiry() != -1 {
			t.Errorf("Expected TimeToExpiry to be -1 for expired k-pak, got %d", kpak.TimeToExpiry())
		}
	})

	t.Run("future k-pak returns correct time to expiry", func(t *testing.T) {
		ttl := int64(300) // 5 minutes
		kpak := NewKpakWithTTL("subject1", "predicate1", "value1", "test-source", 1.0, ttl)

		timeToExpiry := kpak.TimeToExpiry()
		if timeToExpiry < ttl-2 || timeToExpiry > ttl {
			t.Errorf("Expected TimeToExpiry around %d, got %d", ttl, timeToExpiry)
		}
	})
}

func TestKpakTTLSerialization(t *testing.T) {
	t.Run("k-pak with TTL serializes and deserializes correctly", func(t *testing.T) {
		original := NewKpakWithTTL("subject1", "predicate1", "value1", "test-source", 0.95, 3600)

		// Serialize
		data, err := original.ToJSON()
		if err != nil {
			t.Fatalf("Failed to serialize k-pak: %v", err)
		}

		// Deserialize
		restored, err := FromJSON(data)
		if err != nil {
			t.Fatalf("Failed to deserialize k-pak: %v", err)
		}

		// Check all fields
		if restored.Subject != original.Subject {
			t.Errorf("Subject mismatch: expected %s, got %s", original.Subject, restored.Subject)
		}
		if restored.ExpiresAt != original.ExpiresAt {
			t.Errorf("ExpiresAt mismatch: expected %d, got %d", original.ExpiresAt, restored.ExpiresAt)
		}
		if restored.ID != original.ID {
			t.Errorf("ID mismatch: expected %s, got %s", original.ID, restored.ID)
		}
	})
}
