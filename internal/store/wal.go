package store

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Pew-X/sutra/internal/core"
)

// WAL (Write-Ahead Log) provides persistent storage for k-paks.
// It ensures the agent's memory survives restarts. very primitive
type WAL struct {
	filePath string
	file     *os.File
	mutex    sync.Mutex
}

// NewWAL creates a new Write-Ahead Log at the specified path.
func NewWAL(filePath string) (*WAL, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	// Open file for append
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}

	return &WAL{
		filePath: filePath,
		file:     file,
	}, nil
}

// Append writes a k-pak to the log.
func (w *WAL) Append(kpak *core.Kpak) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Serialize k-pak to JSON
	data, err := kpak.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize k-pak: %w", err)
	}

	// Write to file with newline
	_, err = fmt.Fprintf(w.file, "%s\n", data)
	if err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	// Force sync to disk for durability may require more sophisticated handling in production
	return w.file.Sync()
}

// Load reads all k-paks from the log file.
func (w *WAL) Load() ([]*core.Kpak, error) {
	// Open file for reading
	file, err := os.Open(w.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet - that's ok
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open WAL for reading: %w", err)
	}
	defer file.Close()

	var kpaks []*core.Kpak
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Parse k-pak from JSON
		kpak, err := core.FromJSON([]byte(line))
		if err != nil {
			// Log error but continue - don't let one bad line break everything
			fmt.Printf("Warning: failed to parse k-pak at line %d: %v\n", lineNum, err)
			continue
		}

		kpaks = append(kpaks, kpak)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading WAL file: %w", err)
	}

	return kpaks, nil
}

// Close closes the WAL file.
func (w *WAL) Close() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil // Set to nil to make Close idempotent
		return err
	}
	return nil
}

// Stats returns information about the WAL.
func (w *WAL) Stats() (map[string]interface{}, error) {
	info, err := os.Stat(w.filePath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"file_path": w.filePath,
		"file_size": info.Size(),
		"modified":  info.ModTime(),
	}, nil
}
