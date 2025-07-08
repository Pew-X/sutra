package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Kpak represents a knowledge packet - the atomic unit of knowledge in Synapse.
// It follows a Subject-Predicate-Object (SPO) triple model with metadata.
// In future, we may extend this to support more complex relationships or types like property graph possibly
type Kpak struct {
	// Core triple
	Subject   string      `json:"subject"`   // Who/what this is about
	Predicate string      `json:"predicate"` // The relationship/property
	Object    interface{} `json:"object"`    // The value/target

	// Metadata for reconciliation
	Source     string  `json:"source"`     // Origin of this knowledge
	Confidence float32 `json:"confidence"` // Trust level (0.0-1.0)
	Timestamp  int64   `json:"timestamp"`  // When this was created

	// Computed fields for performance
	ID   string `json:"id"`   // Content hash for uniqueness
	SPID string `json:"spid"` // Subject+Predicate hash for indexing
}

// NewKpak creates a new knowledge packet with computed metadata.
func NewKpak(subject, predicate string, object interface{}, source string, confidence float32) *Kpak {
	now := time.Now().Unix()

	kpak := &Kpak{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     source,
		Confidence: confidence,
		Timestamp:  now,
	}

	// Generate content-based ID
	kpak.ID = kpak.generateID()
	kpak.SPID = kpak.generateSPID()

	return kpak
}

// generateID creates a unique hash of the k-pak's content.
func (k *Kpak) generateID() string {
	data := fmt.Sprintf("%s|%s|%v|%s|%f|%d",
		k.Subject, k.Predicate, k.Object, k.Source, k.Confidence, k.Timestamp)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:16] // First 16 chars for readability
}

// GenerateSPID creates a hash of Subject+Predicate for faster indexing.
func (k *Kpak) GenerateSPID() string {
	data := fmt.Sprintf("%s|%s", k.Subject, k.Predicate)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:12] // Shorter for performance
}

// generateSPID creates a hash of Subject+Predicate for fast indexing.
func (k *Kpak) generateSPID() string {
	return k.GenerateSPID()
}

// IsMoreTrustedThan determines if this k-pak should override another.
// Primary rule: Higher confidence wins currently a very demostrative rule
// (in future we may use more complex heuristics).
// This is the primary rule for reconciliation currently.
// Tie-breaker??: More recent timestamp wins
func (k *Kpak) IsMoreTrustedThan(other *Kpak) bool {
	if k.Confidence > other.Confidence {
		return true
	}
	if k.Confidence == other.Confidence {
		return k.Timestamp > other.Timestamp
	}
	return false
}

// ToJSON serializes the k-pak for persistence or network transfer.
func (k *Kpak) ToJSON() ([]byte, error) {
	return json.Marshal(k)
}

// FromJSON deserializes a k-pak from JSON.
func FromJSON(data []byte) (*Kpak, error) {
	var kpak Kpak
	err := json.Unmarshal(data, &kpak)
	if err != nil {
		return nil, err
	}
	return &kpak, nil
}
