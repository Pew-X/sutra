package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"

	v1 "github.com/Pew-X/sutra/api/v1"
	"github.com/Pew-X/sutra/internal/core"
	"github.com/Pew-X/sutra/internal/gossip"
	"github.com/Pew-X/sutra/internal/monitoring"
	"github.com/Pew-X/sutra/internal/reconciliation"
	"github.com/Pew-X/sutra/internal/store"
)

// Config holds the agent config.
type Config struct {
	Host       string   `yaml:"host"`
	GRPCPort   int      `yaml:"grpc_port"`
	GossipPort int      `yaml:"gossip_port"`
	JoinPeers  []string `yaml:"join_peers"`
	LogLevel   string   `yaml:"log_level"`
	WALPath    string   `yaml:"wal_path"`

	// TTL and Garbage Collection settings
	DefaultTTLSeconds int64 `yaml:"default_ttl_seconds"` // Default TTL for k-paks (0 = never expires)
	GCIntervalSeconds int64 `yaml:"gc_interval_seconds"` // How often to run garbage collection
	GCEnabled         bool  `yaml:"gc_enabled"`          // Whether to enable garbage collection
}

// Agent is the main coordinator that manages all mesh components.
type Agent struct {
	v1.UnimplementedSynapseServiceServer

	config    Config
	engine    *reconciliation.Engine
	wal       *store.WAL
	gossip    *gossip.Manager
	metrics   *monitoring.Metrics
	gc        *GarbageCollector
	server    *grpc.Server
	startTime time.Time

	// State
	mutex   sync.RWMutex
	running bool
}

// NewAgent creates a new Synapse agent.
func NewAgent(config Config) (*Agent, error) {
	// Initialize reconciliation engine
	engine := reconciliation.NewEngine()

	// Initialize WAL
	wal, err := store.NewWAL(config.WALPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize WAL: %w", err)
	}

	// Initialize gossip manager
	gossipConfig := &gossip.Config{
		BindAddr:    config.Host,
		BindPort:    config.GossipPort,
		JoinPeers:   config.JoinPeers,
		ClusterName: "synapse-mesh",
	}

	gossipManager, err := gossip.NewManager(gossipConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gossip manager: %w", err)
	}

	// Initialize metrics
	metrics := monitoring.NewMetrics()

	// Initialize garbage collector
	gc := NewGarbageCollector(engine, config.GCIntervalSeconds, config.GCEnabled)

	agent := &Agent{
		config:    config,
		engine:    engine,
		wal:       wal,
		gossip:    gossipManager,
		metrics:   metrics,
		gc:        gc,
		startTime: time.Now(),
	}

	// Set up gossip callback for handling received k-paks
	gossipManager.SetKpakHandler(func(kpak *core.Kpak) bool {
		accepted := agent.engine.Reconcile(kpak)
		if accepted {
			// Persist to WAL
			if err := agent.wal.Append(kpak); err != nil {
				log.Printf("Warning: failed to persist gossiped k-pak to WAL: %v", err)
			}
		}
		agent.metrics.RecordIngest(kpak.Source, accepted)
		return accepted
	})

	return agent, nil
}

// Start starts the agent and all its components.
func (a *Agent) Start() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.running {
		return fmt.Errorf("agent is already running")
	}

	log.Printf("Starting Synapse agent...")

	// Load existing knowledge from WAL
	if err := a.loadFromWAL(); err != nil {
		return fmt.Errorf("failed to load from WAL: %w", err)
	}

	// Start gRPC server
	if err := a.startGRPCServer(); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	// Start gossip manager
	if err := a.gossip.Start(); err != nil {
		return fmt.Errorf("failed to start gossip manager: %w", err)
	}

	// Start garbage collector
	a.gc.Start()

	a.running = true
	log.Printf("Sutra agent started successfully on %s:%d", a.config.Host, a.config.GRPCPort)
	log.Printf("Gossip network active on %s:%d", a.config.Host, a.config.GossipPort)

	return nil
}

// loadFromWAL restores the agent's state from the Write-Ahead Log.
func (a *Agent) loadFromWAL() error {
	log.Printf("Loading knowledge from WAL: %s", a.config.WALPath)

	kpaks, err := a.wal.Load()
	if err != nil {
		return err
	}

	accepted := 0
	for _, kpak := range kpaks {
		if a.engine.Reconcile(kpak) {
			accepted++
		}
	}

	log.Printf("Loaded %d k-paks from WAL, %d accepted as current truth", len(kpaks), accepted)
	return nil
}

// startGRPCServer initializes and starts the gRPC server.
func (a *Agent) startGRPCServer() error {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", a.config.Host, a.config.GRPCPort))
	if err != nil {
		return err
	}

	a.server = grpc.NewServer()
	v1.RegisterSynapseServiceServer(a.server, a)

	go func() {
		if err := a.server.Serve(listen); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	return nil
}

// gracefully shuts down the agent.
func (a *Agent) Shutdown() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.running {
		return nil
	}

	log.Printf("Shutting down Synapse agent...")

	// Stop garbage collector
	if a.gc != nil {
		a.gc.Stop()
	}

	// Stop gossip manager
	if a.gossip != nil {
		a.gossip.Stop()
	}

	// Stop gRPC server
	if a.server != nil {
		a.server.GracefulStop()
	}

	// Close WAL
	if a.wal != nil {
		a.wal.Close()
	}

	a.running = false
	log.Printf("Sutra agent shut down")

	return nil
}

// Implementation of SynapseServiceServer interface

// Ingest handles streaming k-pak ingestion.
func (a *Agent) Ingest(stream v1.SynapseService_IngestServer) error {
	accepted := int32(0)
	rejected := int32(0)
	var errors []string

	for {
		protoKpak, err := stream.Recv()
		if err != nil {
			// End of stream
			break
		}

		// Convert proto k-pak to internal k-pak
		kpak := a.protoToKpak(protoKpak)

		// Try to reconcile
		if a.engine.Reconcile(kpak) {
			// Accepted - persist to WAL
			if err := a.wal.Append(kpak); err != nil {
				errors = append(errors, fmt.Sprintf("failed to persist k-pak: %v", err))
				rejected++
				a.metrics.RecordIngest(kpak.Source, false)
			} else {
				accepted++
				a.metrics.RecordIngest(kpak.Source, true)

				// Broadcast to gossip mesh
				if err := a.gossip.BroadcastKpak(kpak); err != nil {
					log.Printf("Warning: failed to broadcast k-pak to mesh: %v", err)
				}
			}
		} else {
			rejected++
			a.metrics.RecordIngest(kpak.Source, false)
		}
	}

	return stream.SendAndClose(&v1.IngestResponse{
		Accepted: accepted,
		Rejected: rejected,
		Errors:   errors,
	})
}

// query handles k-pak queries.
func (a *Agent) Query(req *v1.QueryRequest, stream v1.SynapseService_QueryServer) error {
	a.metrics.RecordQuery()

	var kpaks []*core.Kpak

	if req.Predicate != nil && *req.Predicate != "" {
		// Specific subject+predicate query
		kpak := a.engine.QueryBySubjectPredicate(req.Subject, *req.Predicate)
		if kpak != nil {
			kpaks = []*core.Kpak{kpak}
		}
	} else {
		// All k-paks for subject
		kpaks = a.engine.QueryBySubject(req.Subject)
	}

	// Stream results
	for _, kpak := range kpaks {
		protoKpak := a.kpakToProto(kpak)
		if err := stream.Send(protoKpak); err != nil {
			return err
		}
	}

	return nil
}

// Health returns the agent's health status.
func (a *Agent) Health(ctx context.Context, req *v1.HealthRequest) (*v1.HealthResponse, error) {
	stats := a.engine.GetStats()
	totalKpaks := int32(stats["total_kpaks"].(int))

	health := a.metrics.GetHealthStatus(totalKpaks)

	return &v1.HealthResponse{
		Status:        health.Status,
		KpakCount:     health.KpakCount,
		UptimeSeconds: health.UptimeSeconds,
	}, nil
}

// GetPeers returns information about mesh peers.
func (a *Agent) GetPeers(ctx context.Context, req *v1.PeersRequest) (*v1.PeersResponse, error) {
	members := a.gossip.GetMembers()
	peers := make([]*v1.PeerInfo, len(members))

	for i, member := range members {
		peers[i] = &v1.PeerInfo{
			Address:  fmt.Sprintf("%s:%d", member.Addr, member.Port),
			Name:     member.Name,
			State:    int32(member.State),
			LastSeen: time.Now().Unix(), // TODO: Get actual last seen time
		}
	}

	return &v1.PeersResponse{
		Peers: peers,
	}, nil
}

// GetMetrics returns agent performance metrics.
func (a *Agent) GetMetrics(ctx context.Context, req *v1.MetricsRequest) (*v1.MetricsResponse, error) {
	stats := a.engine.GetStats()
	totalKpaks := int32(stats["total_kpaks"].(int))
	totalSubjects := int32(stats["total_subjects"].(int))

	metrics := a.metrics.GetMetrics(totalKpaks, totalSubjects)

	return &v1.MetricsResponse{
		TotalKpaks:       metrics.TotalKpaks,
		TotalSubjects:    metrics.TotalSubjects,
		IngestRatePerMin: metrics.IngestRatePerMin,
		QueryRatePerMin:  metrics.QueryRatePerMin,
		UptimeSeconds:    metrics.UptimeSeconds,
		MemoryUsageBytes: metrics.MemoryUsageBytes,
		CpuUsagePercent:  metrics.CPUUsagePercent,
		Version:          metrics.Version,
		ActiveSources:    metrics.ActiveSources,
	}, nil
}

// Helper methods

func (a *Agent) protoToKpak(proto *v1.Kpak) *core.Kpak {
	// Calculate TTL from expires_at field or use default
	var ttlSeconds int64
	if proto.ExpiresAt > 0 {
		now := time.Now().Unix()
		if proto.ExpiresAt > now {
			ttlSeconds = proto.ExpiresAt - now
		} else {
			ttlSeconds = 0 // Already expired, but we'll let reconciliation handle it
		}
	} else if a.config.DefaultTTLSeconds > 0 {
		ttlSeconds = a.config.DefaultTTLSeconds
	}

	// Create k-pak with TTL
	kpak := core.NewKpakWithTTL(
		proto.Subject,
		proto.Predicate,
		proto.Object,
		proto.Source,
		proto.Confidence,
		ttlSeconds,
	)

	// If the proto had specific timestamp and expires_at, preserve them
	if proto.Timestamp > 0 {
		kpak.Timestamp = proto.Timestamp
	}
	if proto.ExpiresAt > 0 {
		kpak.ExpiresAt = proto.ExpiresAt
	}

	// Regenerate IDs if we modified timestamp
	if proto.Timestamp > 0 {
		// Regenerate computed fields manually since we're modifying the struct
		data := fmt.Sprintf("%s|%s|%v|%s|%f|%d",
			kpak.Subject, kpak.Predicate, kpak.Object, kpak.Source, kpak.Confidence, kpak.Timestamp)
		hash := sha256.Sum256([]byte(data))
		kpak.ID = fmt.Sprintf("%x", hash)[:16] // First 16 chars for readability
		kpak.SPID = kpak.GenerateSPID()
	}

	return kpak
}

func (a *Agent) kpakToProto(kpak *core.Kpak) *v1.Kpak {
	return &v1.Kpak{
		Subject:    kpak.Subject,
		Predicate:  kpak.Predicate,
		Object:     fmt.Sprintf("%v", kpak.Object), // Convert to string
		Source:     kpak.Source,
		Confidence: kpak.Confidence,
		Timestamp:  kpak.Timestamp,
		Id:         kpak.ID,
		Spid:       kpak.SPID,
		ExpiresAt:  kpak.ExpiresAt,
	}
}
