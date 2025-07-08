package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pew-X/sutra/internal/core"
)

func TestNewWAL(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	if wal.filePath != walPath {
		t.Fatalf("Expected file path %s, got %s", walPath, wal.filePath)
	}

	// Verify file was created
	if _, err := os.Stat(walPath); os.IsNotExist(err) {
		t.Fatal("WAL file was not created")
	}
}

func TestNewWAL_CreateDirectory(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Use nested directory that doesn't exist
	walPath := filepath.Join(tempDir, "nested", "dir", "test.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL with nested dir: %v", err)
	}
	defer wal.Close()

	// Verify nested directory was created
	if _, err := os.Stat(filepath.Dir(walPath)); os.IsNotExist(err) {
		t.Fatal("Nested directory was not created")
	}
}

func TestWAL_Append(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Create and append a k-pak
	kpak := core.NewKpak("Alice", "age", "25", "TestSource", 0.8)
	err = wal.Append(kpak)
	if err != nil {
		t.Fatalf("Failed to append k-pak: %v", err)
	}

	// Verify file has content
	data, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatalf("Failed to read WAL file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("WAL file is empty after append")
	}

	// Should contain JSON representation
	if !strings.Contains(string(data), "Alice") {
		t.Fatal("WAL file doesn't contain expected k-pak data")
	}
}

func TestWAL_AppendMultiple(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append multiple k-paks
	kpaks := []*core.Kpak{
		core.NewKpak("Alice", "age", "25", "Source1", 0.8),
		core.NewKpak("Bob", "height", "6ft", "Source2", 0.7),
		core.NewKpak("Charlie", "city", "NYC", "Source3", 0.9),
	}

	for _, kpak := range kpaks {
		err = wal.Append(kpak)
		if err != nil {
			t.Fatalf("Failed to append k-pak: %v", err)
		}
	}

	// Verify file has all content
	data, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatalf("Failed to read WAL file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Alice") ||
		!strings.Contains(content, "Bob") ||
		!strings.Contains(content, "Charlie") {
		t.Fatal("WAL file doesn't contain all expected k-paks")
	}

	// Should have 3 lines (one per k-pak)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}
}

func TestWAL_Load_EmptyFile(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Load from empty file
	kpaks, err := wal.Load()
	if err != nil {
		t.Fatalf("Failed to load from empty WAL: %v", err)
	}

	if kpaks != nil {
		t.Fatalf("Expected nil result from empty WAL, got %d k-paks", len(kpaks))
	}
}

func TestWAL_Load_NonexistentFile(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "nonexistent.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Delete the file to simulate nonexistent state
	os.Remove(walPath)

	// Load from nonexistent file should succeed with nil result
	kpaks, err := wal.Load()
	if err != nil {
		t.Fatalf("Failed to load from nonexistent WAL: %v", err)
	}

	if kpaks != nil {
		t.Fatalf("Expected nil result from nonexistent WAL, got %d k-paks", len(kpaks))
	}
}

func TestWAL_LoadAfterAppend(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append some k-paks
	originalKpaks := []*core.Kpak{
		core.NewKpak("Alice", "age", "25", "Source1", 0.8),
		core.NewKpak("Bob", "height", "6ft", "Source2", 0.7),
		core.NewKpak("Charlie", "city", "NYC", "Source3", 0.9),
	}

	for _, kpak := range originalKpaks {
		err = wal.Append(kpak)
		if err != nil {
			t.Fatalf("Failed to append k-pak: %v", err)
		}
	}

	// Load them back
	loadedKpaks, err := wal.Load()
	if err != nil {
		t.Fatalf("Failed to load k-paks: %v", err)
	}

	if len(loadedKpaks) != len(originalKpaks) {
		t.Fatalf("Expected %d k-paks, got %d", len(originalKpaks), len(loadedKpaks))
	}

	// Verify content (note: loaded k-paks will have different timestamps/IDs)
	subjects := make(map[string]bool)
	for _, kpak := range loadedKpaks {
		subjects[kpak.Subject] = true
	}

	if !subjects["Alice"] || !subjects["Bob"] || !subjects["Charlie"] {
		t.Fatal("Missing expected subjects in loaded k-paks")
	}
}

func TestWAL_LoadWithCorruptedLine(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")

	// Write some valid and invalid JSON to file
	content := `{"subject":"Alice","predicate":"age","object":"25","source":"Source1","confidence":0.8,"timestamp":1234567890,"id":"abc123","spid":"def456"}
this is not valid json
{"subject":"Bob","predicate":"height","object":"6ft","source":"Source2","confidence":0.7,"timestamp":1234567891,"id":"ghi789","spid":"jkl012"}

{"subject":"Charlie","predicate":"city","object":"NYC","source":"Source3","confidence":0.9,"timestamp":1234567892,"id":"mno345","spid":"pqr678"}
`

	err = os.WriteFile(walPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Load should succeed but skip corrupted line
	kpaks, err := wal.Load()
	if err != nil {
		t.Fatalf("Failed to load k-paks: %v", err)
	}

	// Should have 3 valid k-paks (corrupted line and empty line skipped)
	if len(kpaks) != 3 {
		t.Fatalf("Expected 3 k-paks, got %d", len(kpaks))
	}

	subjects := make(map[string]bool)
	for _, kpak := range kpaks {
		subjects[kpak.Subject] = true
	}

	if !subjects["Alice"] || !subjects["Bob"] || !subjects["Charlie"] {
		t.Fatal("Missing expected subjects in loaded k-paks")
	}
}

func TestWAL_Stats(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append a k-pak
	kpak := core.NewKpak("Alice", "age", "25", "TestSource", 0.8)
	err = wal.Append(kpak)
	if err != nil {
		t.Fatalf("Failed to append k-pak: %v", err)
	}

	// Get stats
	stats, err := wal.Stats()
	if err != nil {
		t.Fatalf("Failed to get WAL stats: %v", err)
	}

	if stats["file_path"] != walPath {
		t.Fatalf("Expected file_path %s, got %v", walPath, stats["file_path"])
	}

	if stats["file_size"].(int64) <= 0 {
		t.Fatal("Expected positive file size")
	}

	if _, ok := stats["modified"]; !ok {
		t.Fatal("Expected modified timestamp in stats")
	}
}

func TestWAL_ConcurrentAppend(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Test concurrent appends
	const numGoroutines = 10
	const numAppendsPerGoroutine = 50

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numAppendsPerGoroutine; j++ {
				kpak := core.NewKpak(
					fmt.Sprintf("Subject%d_%d", id, j),
					"property",
					fmt.Sprintf("value%d_%d", id, j),
					fmt.Sprintf("Source%d", id),
					0.5,
				)
				if err := wal.Append(kpak); err != nil {
					t.Errorf("Failed to append k-pak: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Load and verify all k-paks were written
	kpaks, err := wal.Load()
	if err != nil {
		t.Fatalf("Failed to load k-paks: %v", err)
	}

	expectedCount := numGoroutines * numAppendsPerGoroutine
	if len(kpaks) != expectedCount {
		t.Fatalf("Expected %d k-paks, got %d", expectedCount, len(kpaks))
	}
}

func TestWAL_Close(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// Close should succeed
	err = wal.Close()
	if err != nil {
		t.Fatalf("Failed to close WAL: %v", err)
	}

	// Second close should also succeed (idempotent)
	err = wal.Close()
	if err != nil {
		t.Fatalf("Second close failed: %v", err)
	}
}

func TestWAL_PersistenceAcrossInstances(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")

	// Create first WAL instance and append data
	wal1, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create first WAL: %v", err)
	}

	kpak1 := core.NewKpak("Alice", "age", "25", "Source1", 0.8)
	err = wal1.Append(kpak1)
	if err != nil {
		t.Fatalf("Failed to append to first WAL: %v", err)
	}

	wal1.Close()

	// Create second WAL instance and verify data persistence
	wal2, err := NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create second WAL: %v", err)
	}
	defer wal2.Close()

	// Append more data
	kpak2 := core.NewKpak("Bob", "height", "6ft", "Source2", 0.7)
	err = wal2.Append(kpak2)
	if err != nil {
		t.Fatalf("Failed to append to second WAL: %v", err)
	}

	// Load all data
	kpaks, err := wal2.Load()
	if err != nil {
		t.Fatalf("Failed to load from second WAL: %v", err)
	}

	if len(kpaks) != 2 {
		t.Fatalf("Expected 2 k-paks, got %d", len(kpaks))
	}

	subjects := make(map[string]bool)
	for _, kpak := range kpaks {
		subjects[kpak.Subject] = true
	}

	if !subjects["Alice"] || !subjects["Bob"] {
		t.Fatal("Missing expected subjects in persisted data")
	}
}
