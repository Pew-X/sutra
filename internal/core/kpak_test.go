package core

import (
	"testing"
	"time"
)

func TestNewKpak(t *testing.T) {
	tests := []struct {
		name        string
		subject     string
		predicate   string
		object      interface{}
		source      string
		confidence  float32
		expectValid bool
	}{
		{
			name:        "Valid string object",
			subject:     "user123",
			predicate:   "name",
			object:      "John Doe",
			source:      "test_source",
			confidence:  0.9,
			expectValid: true,
		},
		{
			name:        "Valid number object",
			subject:     "user123",
			predicate:   "age",
			object:      25,
			source:      "test_source",
			confidence:  1.0,
			expectValid: true,
		},
		{
			name:        "Empty subject",
			subject:     "",
			predicate:   "name",
			object:      "John Doe",
			source:      "test_source",
			confidence:  0.9,
			expectValid: true, // System allows empty subjects
		},
		{
			name:        "Zero confidence",
			subject:     "user123",
			predicate:   "name",
			object:      "John Doe",
			source:      "test_source",
			confidence:  0.0,
			expectValid: true,
		},
		{
			name:        "Max confidence",
			subject:     "user123",
			predicate:   "name",
			object:      "John Doe",
			source:      "test_source",
			confidence:  1.0,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kpak := NewKpak(tt.subject, tt.predicate, tt.object, tt.source, tt.confidence)

			// Check basic fields
			if kpak.Subject != tt.subject {
				t.Errorf("Expected subject %s, got %s", tt.subject, kpak.Subject)
			}
			if kpak.Predicate != tt.predicate {
				t.Errorf("Expected predicate %s, got %s", tt.predicate, kpak.Predicate)
			}
			if kpak.Object != tt.object {
				t.Errorf("Expected object %v, got %v", tt.object, kpak.Object)
			}
			if kpak.Source != tt.source {
				t.Errorf("Expected source %s, got %s", tt.source, kpak.Source)
			}
			if kpak.Confidence != tt.confidence {
				t.Errorf("Expected confidence %f, got %f", tt.confidence, kpak.Confidence)
			}

			// Check computed fields
			if kpak.ID == "" {
				t.Error("Expected non-empty ID")
			}
			if kpak.SPID == "" {
				t.Error("Expected non-empty SPID")
			}
			if kpak.Timestamp == 0 {
				t.Error("Expected non-zero timestamp")
			}

			// Check timestamp is recent
			now := time.Now().Unix()
			if kpak.Timestamp > now || kpak.Timestamp < now-10 {
				t.Errorf("Timestamp %d seems invalid (now: %d)", kpak.Timestamp, now)
			}
		})
	}
}

func TestKpakIDGeneration(t *testing.T) {
	// Test that different content generates different IDs
	kpak1 := NewKpak("subject", "predicate", "object", "source1", 0.9)
	kpak2 := NewKpak("subject", "predicate", "object", "source2", 0.9)

	// IDs should be different due to different sources
	if kpak1.ID == kpak2.ID {
		t.Errorf("Expected different IDs for different sources. ID1: %s, ID2: %s", kpak1.ID, kpak2.ID)
	}

	// SPIDs should be same (subject+predicate)
	if kpak1.SPID != kpak2.SPID {
		t.Error("Expected same SPIDs for same subject+predicate")
	}

	// Test that identical content generates identical IDs (when using manual timestamps)
	testKpak1 := &Kpak{
		Subject:    "test",
		Predicate:  "type",
		Object:     "value",
		Source:     "test",
		Confidence: 0.5,
		Timestamp:  1234567890,
	}
	testKpak1.ID = testKpak1.generateID()

	testKpak2 := &Kpak{
		Subject:    "test",
		Predicate:  "type",
		Object:     "value",
		Source:     "test",
		Confidence: 0.5,
		Timestamp:  1234567890,
	}
	testKpak2.ID = testKpak2.generateID()

	// These should have identical IDs since all content is identical
	if testKpak1.ID != testKpak2.ID {
		t.Errorf("Expected identical IDs for identical content. ID1: %s, ID2: %s", testKpak1.ID, testKpak2.ID)
	}

	// Test manual ID generation
	testKpak := &Kpak{
		Subject:    "test",
		Predicate:  "type",
		Object:     "value",
		Source:     "test",
		Confidence: 0.5,
		Timestamp:  1234567890,
	}

	id1 := testKpak.generateID()
	id2 := testKpak.generateID()

	if id1 != id2 {
		t.Error("Expected consistent ID generation")
	}

	if len(id1) != 16 {
		t.Errorf("Expected ID length 16, got %d", len(id1))
	}
}

func TestKpakSPIDGeneration(t *testing.T) {
	tests := []struct {
		subject1, predicate1 string
		subject2, predicate2 string
		expectSame           bool
	}{
		{"user", "name", "user", "name", true},
		{"user", "name", "user", "age", false},
		{"user1", "name", "user2", "name", false},
		{"", "test", "", "test", true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			kpak1 := &Kpak{Subject: tt.subject1, Predicate: tt.predicate1}
			kpak2 := &Kpak{Subject: tt.subject2, Predicate: tt.predicate2}

			spid1 := kpak1.GenerateSPID()
			spid2 := kpak2.GenerateSPID()

			if tt.expectSame && spid1 != spid2 {
				t.Errorf("Expected same SPIDs, got %s and %s", spid1, spid2)
			}
			if !tt.expectSame && spid1 == spid2 {
				t.Errorf("Expected different SPIDs, both got %s", spid1)
			}

			// Check SPID length
			if len(spid1) != 12 {
				t.Errorf("Expected SPID length 12, got %d", len(spid1))
			}
		})
	}
}

func TestKpakIsMoreTrustedThan(t *testing.T) {
	baseTime := time.Now().Unix()

	tests := []struct {
		name     string
		k1       *Kpak
		k2       *Kpak
		expected bool
	}{
		{
			name:     "Higher confidence wins",
			k1:       &Kpak{Confidence: 0.9, Timestamp: baseTime},
			k2:       &Kpak{Confidence: 0.8, Timestamp: baseTime},
			expected: true,
		},
		{
			name:     "Lower confidence loses",
			k1:       &Kpak{Confidence: 0.7, Timestamp: baseTime},
			k2:       &Kpak{Confidence: 0.8, Timestamp: baseTime},
			expected: false,
		},
		{
			name:     "Same confidence, newer timestamp wins",
			k1:       &Kpak{Confidence: 0.8, Timestamp: baseTime + 10},
			k2:       &Kpak{Confidence: 0.8, Timestamp: baseTime},
			expected: true,
		},
		{
			name:     "Same confidence, older timestamp loses",
			k1:       &Kpak{Confidence: 0.8, Timestamp: baseTime},
			k2:       &Kpak{Confidence: 0.8, Timestamp: baseTime + 10},
			expected: false,
		},
		{
			name:     "Identical confidence and timestamp",
			k1:       &Kpak{Confidence: 0.8, Timestamp: baseTime},
			k2:       &Kpak{Confidence: 0.8, Timestamp: baseTime},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.k1.IsMoreTrustedThan(tt.k2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestKpakJSON(t *testing.T) {
	original := NewKpak("user123", "name", "John Doe", "test_source", 0.9)

	// Test ToJSON
	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Test FromJSON
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// Compare fields
	if restored.Subject != original.Subject {
		t.Errorf("Subject mismatch: %s vs %s", restored.Subject, original.Subject)
	}
	if restored.Predicate != original.Predicate {
		t.Errorf("Predicate mismatch: %s vs %s", restored.Predicate, original.Predicate)
	}
	if restored.Source != original.Source {
		t.Errorf("Source mismatch: %s vs %s", restored.Source, original.Source)
	}
	if restored.Confidence != original.Confidence {
		t.Errorf("Confidence mismatch: %f vs %f", restored.Confidence, original.Confidence)
	}
	if restored.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: %d vs %d", restored.Timestamp, original.Timestamp)
	}
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: %s vs %s", restored.ID, original.ID)
	}
	if restored.SPID != original.SPID {
		t.Errorf("SPID mismatch: %s vs %s", restored.SPID, original.SPID)
	}
}

func TestKpakJSONWithInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"Invalid JSON", `{"invalid": json}`},
		{"Empty string", ""},
		{"Non-JSON", "not json at all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromJSON([]byte(tt.data))
			if err == nil {
				t.Error("Expected error for invalid JSON")
			}
		})
	}
}

// Benchmark tests
func BenchmarkNewKpak(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewKpak("subject", "predicate", "object", "source", 0.9)
	}
}

func BenchmarkKpakIDGeneration(b *testing.B) {
	kpak := NewKpak("subject", "predicate", "object", "source", 0.9)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kpak.generateID()
	}
}

func BenchmarkKpakSPIDGeneration(b *testing.B) {
	kpak := NewKpak("subject", "predicate", "object", "source", 0.9)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kpak.GenerateSPID()
	}
}

func BenchmarkKpakJSON(b *testing.B) {
	kpak := NewKpak("subject", "predicate", "object", "source", 0.9)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		data, _ := kpak.ToJSON()
		FromJSON(data)
	}
}
