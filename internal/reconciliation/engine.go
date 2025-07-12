package reconciliation

import (
	"sync"

	"github.com/Pew-X/sutra/internal/core"
)

// Engine is the "brain" that decides what is true.
// It manages the reconciliation logic and maintains the current state of truth.
type Engine struct {
	// truthStore holds the currently accepted truths, indexed by SPID
	truthStore map[string]*core.Kpak
	// subjectIndex allows fast lookup by subject
	subjectIndex map[string]map[string]struct{} // subject -> set of SPIDs
	mutex        sync.RWMutex
}

// NewEngine creates a new reconciliation engine.
func NewEngine() *Engine {
	return &Engine{
		truthStore:   make(map[string]*core.Kpak),
		subjectIndex: make(map[string]map[string]struct{}),
	}
}

// Reconcile processes a new k-pak and determines if it should be accepted.
// Returns true if the k-pak was accepted (new truth), false if rejected.
func (e *Engine) Reconcile(kpak *core.Kpak) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Check if we already have a k-pak for this Subject+Predicate
	existing, exists := e.truthStore[kpak.SPID]

	if !exists {
		// New knowledge - accept it
		e.acceptKpak(kpak)
		return true
	}

	// Conflict resolution: check if new k-pak is more trusted
	if kpak.IsMoreTrustedThan(existing) {
		// Replace existing with new k-pak
		e.acceptKpak(kpak)
		return true
	}

	// Reject the k-pak - existing truth is more trusted
	return false
}

// acceptKpak stores a k-pak as accepted truth and updates indices.
func (e *Engine) acceptKpak(kpak *core.Kpak) {
	// Store in truth store
	e.truthStore[kpak.SPID] = kpak

	// Update subject index
	if _, exists := e.subjectIndex[kpak.Subject]; !exists {
		e.subjectIndex[kpak.Subject] = make(map[string]struct{})
	}
	e.subjectIndex[kpak.Subject][kpak.SPID] = struct{}{}
}

// QueryBySubject returns all accepted k-paks for a given subject. simple full text matching , may require semantics in future
func (e *Engine) QueryBySubject(subject string) []*core.Kpak {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	spids, exists := e.subjectIndex[subject]
	if !exists {
		return nil
	}

	var results []*core.Kpak
	for spid := range spids {
		if kpak, exists := e.truthStore[spid]; exists {
			results = append(results, kpak)
		}
	}

	return results
}

// QueryBySubjectPredicate returns the accepted k-pak for a specific subject+predicate. may require semantics
func (e *Engine) QueryBySubjectPredicate(subject, predicate string) *core.Kpak {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Generate SPID for lookup
	tempKpak := &core.Kpak{Subject: subject, Predicate: predicate}
	spid := tempKpak.GenerateSPID()

	return e.truthStore[spid]
}

// GetAllTruths returns all currently accepted k-paks.
func (e *Engine) GetAllTruths() []*core.Kpak {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	var results []*core.Kpak
	for _, kpak := range e.truthStore {
		results = append(results, kpak)
	}

	return results
}

// GetStats returns statistics about the current state.
func (e *Engine) GetStats() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return map[string]interface{}{
		"total_kpaks":    len(e.truthStore),
		"total_subjects": len(e.subjectIndex),
	}
}

// RemoveExpiredKpaks removes all expired k-paks from the truth store.
// Returns the number of k-paks that were removed.
func (e *Engine) RemoveExpiredKpaks() int {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var expiredSPIDs []string

	// Find all expired k-paks
	for spid, kpak := range e.truthStore {
		if kpak.IsExpired() {
			expiredSPIDs = append(expiredSPIDs, spid)
		}
	}

	// Remove expired k-paks from truth store and update indices
	for _, spid := range expiredSPIDs {
		kpak := e.truthStore[spid]

		// Remove from truth store
		delete(e.truthStore, spid)

		// Update subject index
		if spidSet, exists := e.subjectIndex[kpak.Subject]; exists {
			delete(spidSet, spid)
			// If this was the last SPID for this subject, remove the subject entry
			if len(spidSet) == 0 {
				delete(e.subjectIndex, kpak.Subject)
			}
		}
	}

	return len(expiredSPIDs)
}
