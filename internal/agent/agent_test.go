package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/Pew-X/sutra/api/v1"
	"github.com/Pew-X/sutra/internal/core"
)

func TestNewAgent(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0, // Let OS choose port
		GossipPort: 0, // Let OS choose port
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    filepath.Join(tempDir, "test.log"),
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agent == nil {
		t.Fatal("NewAgent returned nil")
	}

	if agent.config.Host != config.Host {
		t.Fatal("Config not set correctly")
	}

	if agent.engine == nil {
		t.Fatal("Reconciliation engine not initialized")
	}

	if agent.wal == nil {
		t.Fatal("WAL not initialized")
	}

	if agent.gossip == nil {
		t.Fatal("Gossip manager not initialized")
	}

	if agent.metrics == nil {
		t.Fatal("Metrics not initialized")
	}

	if agent.running {
		t.Fatal("Agent should not be running initially")
	}
}

func TestNewAgent_InvalidWALPath(t *testing.T) {
	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0,
		GossipPort: 0,
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    "/invalid/path/that/does/not/exist.log",
	}

	// On Windows, this path might actually be valid, so let's use a clearly invalid one
	if os.PathSeparator == '\\' {
		config.WALPath = `\\invalid\path\that\does\not\exist.log`
	}

	agent, err := NewAgent(config)
	if err == nil {
		t.Fatal("Expected error for invalid WAL path")
	}

	if agent != nil {
		t.Fatal("Agent should be nil when creation fails")
	}
}

func TestAgent_StartShutdown(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0, // Let OS choose port
		GossipPort: 0, // Let OS choose port
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    filepath.Join(tempDir, "test.log"),
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Start the agent
	err = agent.Start()
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	if !agent.running {
		t.Fatal("Agent should be running after start")
	}

	// Shutdown the agent
	err = agent.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown agent: %v", err)
	}

	if agent.running {
		t.Fatal("Agent should not be running after shutdown")
	}
}

func TestAgent_StartTwice(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0,
		GossipPort: 0,
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    filepath.Join(tempDir, "test.log"),
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Start first time
	err = agent.Start()
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}
	defer agent.Shutdown()

	// Start second time should fail
	err = agent.Start()
	if err == nil {
		t.Fatal("Starting agent twice should return error")
	}
}

func TestAgent_ShutdownWithoutStart(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0,
		GossipPort: 0,
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    filepath.Join(tempDir, "test.log"),
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Shutdown without start should succeed
	err = agent.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown without start should succeed: %v", err)
	}
}

func TestAgent_Health(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0,
		GossipPort: 0,
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    filepath.Join(tempDir, "test.log"),
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	err = agent.Start()
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}
	defer agent.Shutdown()

	ctx := context.Background()
	req := &v1.HealthRequest{}

	resp, err := agent.Health(ctx, req)
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Status == "" {
		t.Fatal("Status should not be empty")
	}

	if resp.UptimeSeconds < 0 {
		t.Fatal("Uptime should be non-negative")
	}

	if resp.KpakCount < 0 {
		t.Fatal("K-pak count should be non-negative")
	}
}

func TestAgent_GetMetrics(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0,
		GossipPort: 0,
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    filepath.Join(tempDir, "test.log"),
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	err = agent.Start()
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}
	defer agent.Shutdown()

	ctx := context.Background()
	req := &v1.MetricsRequest{}

	resp, err := agent.GetMetrics(ctx, req)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.TotalKpaks < 0 {
		t.Fatal("Total k-paks should be non-negative")
	}

	if resp.UptimeSeconds < 0 {
		t.Fatal("Uptime should be non-negative")
	}

	if resp.MemoryUsageBytes < 0 {
		t.Fatal("Memory usage should be non-negative")
	}

	if resp.Version == "" {
		t.Fatal("Version should not be empty")
	}
}

func TestAgent_GetPeers(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0,
		GossipPort: 0,
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    filepath.Join(tempDir, "test.log"),
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	err = agent.Start()
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}
	defer agent.Shutdown()

	ctx := context.Background()
	req := &v1.PeersRequest{}

	resp, err := agent.GetPeers(ctx, req)
	if err != nil {
		t.Fatalf("Failed to get peers: %v", err)
	}

	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	// Should have at least the local node
	if len(resp.Peers) < 1 {
		t.Fatal("Should have at least one peer (local node)")
	}

	peer := resp.Peers[0]
	if peer.Address == "" {
		t.Fatal("Peer address should not be empty")
	}

	if peer.Name == "" {
		t.Fatal("Peer name should not be empty")
	}
}

func TestAgent_LoadFromWAL(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "test.log")

	// Pre-populate WAL with some data
	kpak1 := core.NewKpak("Alice", "age", "25", "Source1", 0.8)
	kpak2 := core.NewKpak("Bob", "height", "6ft", "Source2", 0.7)

	// Write k-paks to WAL file manually
	walContent := fmt.Sprintf("%s\n%s\n",
		mustMarshal(kpak1),
		mustMarshal(kpak2))

	err = os.WriteFile(walPath, []byte(walContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write WAL file: %v", err)
	}

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0,
		GossipPort: 0,
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    walPath,
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	err = agent.Start()
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}
	defer agent.Shutdown()

	// Verify data was loaded by checking engine state
	aliceKpaks := agent.engine.QueryBySubject("Alice")
	if len(aliceKpaks) != 1 {
		t.Fatalf("Expected 1 k-pak for Alice from WAL, got %d", len(aliceKpaks))
	}

	if aliceKpaks[0].Subject != "Alice" || aliceKpaks[0].Predicate != "age" || aliceKpaks[0].Object != "25" {
		t.Fatal("Loaded k-pak doesn't match expected data")
	}

	bobKpaks := agent.engine.QueryBySubject("Bob")
	if len(bobKpaks) != 1 {
		t.Fatalf("Expected 1 k-pak for Bob from WAL, got %d", len(bobKpaks))
	}
}

func TestAgent_ProtoConversion(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0,
		GossipPort: 0,
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    filepath.Join(tempDir, "test.log"),
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test proto to kpak conversion
	protoKpak := &v1.Kpak{
		Subject:    "Alice",
		Predicate:  "age",
		Object:     "25",
		Source:     "TestSource",
		Confidence: 0.8,
	}

	kpak := agent.protoToKpak(protoKpak)
	if kpak.Subject != "Alice" {
		t.Fatalf("Expected subject 'Alice', got '%s'", kpak.Subject)
	}
	if kpak.Predicate != "age" {
		t.Fatalf("Expected predicate 'age', got '%s'", kpak.Predicate)
	}
	if kpak.Object != "25" {
		t.Fatalf("Expected object '25', got '%s'", kpak.Object)
	}
	if kpak.Source != "TestSource" {
		t.Fatalf("Expected source 'TestSource', got '%s'", kpak.Source)
	}
	if kpak.Confidence != 0.8 {
		t.Fatalf("Expected confidence 0.8, got %f", kpak.Confidence)
	}

	// Verify ID and SPID are generated
	if kpak.ID == "" {
		t.Fatal("ID should be generated")
	}
	if kpak.SPID == "" {
		t.Fatal("SPID should be generated")
	}

	// Test kpak to proto conversion
	backToProto := agent.kpakToProto(kpak)
	if backToProto.Subject != kpak.Subject {
		t.Fatal("Subject mismatch in reverse conversion")
	}
	if backToProto.Predicate != kpak.Predicate {
		t.Fatal("Predicate mismatch in reverse conversion")
	}
	if backToProto.Object != fmt.Sprintf("%v", kpak.Object) {
		t.Fatal("Object mismatch in reverse conversion")
	}
	if backToProto.Source != kpak.Source {
		t.Fatal("Source mismatch in reverse conversion")
	}
	if backToProto.Confidence != kpak.Confidence {
		t.Fatal("Confidence mismatch in reverse conversion")
	}
	if backToProto.Id != kpak.ID {
		t.Fatal("ID mismatch in reverse conversion")
	}
	if backToProto.Spid != kpak.SPID {
		t.Fatal("SPID mismatch in reverse conversion")
	}
}

func TestAgent_ConflictResolution(t *testing.T) {
	// Create temporary WAL directory
	tempDir, err := os.MkdirTemp("", "agent_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := Config{
		Host:       "127.0.0.1",
		GRPCPort:   0,
		GossipPort: 0,
		JoinPeers:  []string{},
		LogLevel:   "INFO",
		WALPath:    filepath.Join(tempDir, "test.log"),
	}

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Add k-pak with lower confidence directly to engine
	kpak1 := core.NewKpak("Alice", "age", "25", "Source1", 0.5)
	accepted1 := agent.engine.Reconcile(kpak1)
	if !accepted1 {
		t.Fatal("First k-pak should be accepted")
	}

	// Add conflicting k-pak with higher confidence
	kpak2 := core.NewKpak("Alice", "age", "26", "Source2", 0.8)
	accepted2 := agent.engine.Reconcile(kpak2)
	if !accepted2 {
		t.Fatal("Higher confidence k-pak should be accepted")
	}

	// Query should return the higher confidence fact
	result := agent.engine.QueryBySubjectPredicate("Alice", "age")
	if result == nil {
		t.Fatal("Should find k-pak for Alice.age")
	}
	if result.Object != "26" {
		t.Fatalf("Expected object '26' (higher confidence), got '%s'", result.Object)
	}

	// Add k-pak with even lower confidence - should be rejected
	kpak3 := core.NewKpak("Alice", "age", "27", "Source3", 0.3)
	accepted3 := agent.engine.Reconcile(kpak3)
	if accepted3 {
		t.Fatal("Lower confidence k-pak should be rejected")
	}

	// Query should still return the previous fact
	result = agent.engine.QueryBySubjectPredicate("Alice", "age")
	if result.Object != "26" {
		t.Fatalf("Expected object '26' (should remain unchanged), got '%s'", result.Object)
	}
}

// Helper function to marshal k-pak to JSON
func mustMarshal(kpak *core.Kpak) string {
	data, err := kpak.ToJSON()
	if err != nil {
		panic(err)
	}
	return string(data)
}
