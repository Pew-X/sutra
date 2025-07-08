package gossip

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"

	"github.com/Pew-X/sutra/internal/core"
)

// Manager handles peer-to-peer gossip networking for the Synapse mesh. based on gossip protocol
type Manager struct {
	config       *Config
	memberlist   *memberlist.Memberlist
	delegate     *synapseDelegate
	eventHandler *synapseEventDelegate

	// Callbacks
	onKpakReceived func(*core.Kpak) bool // Returns true if k-pak was accepted

	mutex   sync.RWMutex
	running bool
}

// Config holds gossip networking configuration.
type Config struct {
	BindAddr    string   // Local bind address
	BindPort    int      // Local gossip port
	JoinPeers   []string // List of peers to join
	ClusterName string   // Cluster identifier
}

// synapseDelegate handles memberlist delegation callbacks.
type synapseDelegate struct {
	manager *Manager
}

// synapseEventDelegate handles member join/leave events.
type synapseEventDelegate struct {
	manager *Manager
}

// NewManager creates a new gossip manager.
func NewManager(config *Config) (*Manager, error) {
	manager := &Manager{
		config: config,
	}

	// Create delegates
	manager.delegate = &synapseDelegate{manager: manager}
	manager.eventHandler = &synapseEventDelegate{manager: manager}

	return manager, nil
}

// Start initializes and starts the gossip protocol.
func (m *Manager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return fmt.Errorf("gossip manager already running")
	}

	// Configure memberlist
	mlConfig := memberlist.DefaultLANConfig()
	mlConfig.Name = fmt.Sprintf("synapse-%s-%d", m.config.BindAddr, m.config.BindPort)
	mlConfig.BindAddr = m.config.BindAddr
	mlConfig.BindPort = m.config.BindPort
	mlConfig.AdvertiseAddr = m.config.BindAddr
	mlConfig.AdvertisePort = m.config.BindPort
	mlConfig.Delegate = m.delegate
	mlConfig.Events = m.eventHandler

	// Reduce log verbose
	mlConfig.Logger = log.New(log.Writer(), "[GOSSIP] ", log.LstdFlags)

	// Create memberlist
	ml, err := memberlist.Create(mlConfig)
	if err != nil {
		return fmt.Errorf("failed to create memberlist: %w", err)
	}

	m.memberlist = ml

	// Join existing cluster if peers specified
	if len(m.config.JoinPeers) > 0 {
		log.Printf("Joining gossip mesh with peers: %v", m.config.JoinPeers)
		joined, err := m.memberlist.Join(m.config.JoinPeers)
		if err != nil {
			log.Printf("Warning: failed to join some peers: %v", err)
		}
		log.Printf("Successfully joined %d peers", joined)
	}

	m.running = true
	log.Printf("Gossip manager started on %s:%d", m.config.BindAddr, m.config.BindPort)

	return nil
}

// gracefully shuts down the gossip protocol.
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return nil
	}

	if m.memberlist != nil {
		if err := m.memberlist.Leave(time.Second * 5); err != nil {
			log.Printf("Warning: error leaving cluster: %v", err)
		}
		if err := m.memberlist.Shutdown(); err != nil {
			log.Printf("Warning: error shutting down memberlist: %v", err)
		}
	}

	m.running = false
	log.Printf("Gossip manager stopped")

	return nil
}

// BroadcastKpak broadcasts a k-pak to all peers in the mesh.
func (m *Manager) BroadcastKpak(kpak *core.Kpak) error {
	if !m.running || m.memberlist == nil {
		return fmt.Errorf("gossip manager not running")
	}

	// Serialize k-pak
	data, err := json.Marshal(kpak)
	if err != nil {
		return fmt.Errorf("failed to serialize k-pak: %w", err)
	}

	// Create gossip message
	msg := &GossipMessage{
		Type:    "kpak",
		Payload: data,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize gossip message: %w", err)
	}

	// Broadcast to cluster
	for _, member := range m.memberlist.Members() {
		if member.Name != m.memberlist.LocalNode().Name {
			if err := m.memberlist.SendBestEffort(member, msgData); err != nil {
				log.Printf("Warning: failed to send message to %s: %v", member.Name, err)
			}
		}
	}

	return nil
}

// SetKpakHandler sets the callback for handling received k-paks.
func (m *Manager) SetKpakHandler(handler func(*core.Kpak) bool) {
	m.onKpakReceived = handler
}

// GetMembers returns information about cluster members.
func (m *Manager) GetMembers() []MemberInfo {
	if !m.running || m.memberlist == nil {
		return nil
	}

	members := m.memberlist.Members()
	result := make([]MemberInfo, len(members))

	for i, member := range members {
		result[i] = MemberInfo{
			Name:  member.Name,
			Addr:  member.Addr.String(),
			Port:  member.Port,
			State: int(member.State),
		}
	}

	return result
}

// MemberInfo represents information about a cluster member.
type MemberInfo struct {
	Name  string `json:"name"`
	Addr  string `json:"addr"`
	Port  uint16 `json:"port"`
	State int    `json:"state"`
}

// GossipMessage represents a message sent through the gossip protocol.
type GossipMessage struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}

// Memberlist delegate implementation

// NodeMeta returns metadata about this node.
func (d *synapseDelegate) NodeMeta(limit int) []byte {
	meta := map[string]interface{}{
		"type":    "synapse-agent",
		"version": "1.0.0",
		"started": time.Now().Unix(),
	}

	data, _ := json.Marshal(meta)
	if len(data) > limit {
		return data[:limit]
	}
	return data
}

// NotifyMsg is called when a message is received from the network.
func (d *synapseDelegate) NotifyMsg(data []byte) {
	var msg GossipMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Warning: failed to unmarshal gossip message: %v", err)
		return
	}

	switch msg.Type {
	case "kpak":
		d.handleKpakMessage(msg.Payload)
	default:
		log.Printf("Warning: unknown gossip message type: %s", msg.Type)
	}
}

// handleKpakMessage processes a received k-pak from the gossip network.
func (d *synapseDelegate) handleKpakMessage(payload []byte) {
	var kpak core.Kpak
	if err := json.Unmarshal(payload, &kpak); err != nil {
		log.Printf("Warning: failed to unmarshal k-pak from gossip: %v", err)
		return
	}

	// Call the handler if set
	if d.manager.onKpakReceived != nil {
		accepted := d.manager.onKpakReceived(&kpak)
		if accepted {
			log.Printf("Gossip: accepted k-pak %s from network", kpak.ID)
		}
	}
}

// GetBroadcasts returns messages to be broadcast.
func (d *synapseDelegate) GetBroadcasts(overhead, limit int) [][]byte {
	// We use SendBestEffort for immediate broadcasting

	return nil
}

// LocalState returns the local state to be sent to joining nodes.
func (d *synapseDelegate) LocalState(join bool) []byte {
	// For now, we don't send state during joins
	// In V2, this could include a snapshot of current knowledge. requires more brainstorming
	return nil
}

// MergeRemoteState merges remote state with local state.
func (d *synapseDelegate) MergeRemoteState(buf []byte, join bool) {
	// For now, we don't handle state merging during joins
	// In V2, this could sync knowledge snapshots or merkle truths (I donno if that made sense?). requires more brainstorming
}

// Event delegate implementation

// NotifyJoin is called when a node joins the cluster.
func (e *synapseEventDelegate) NotifyJoin(node *memberlist.Node) {
	log.Printf("Gossip: node joined cluster: %s (%s:%d)", node.Name, node.Addr, node.Port)
}

// NotifyLeave is called when a node leaves the cluster.
func (e *synapseEventDelegate) NotifyLeave(node *memberlist.Node) {
	log.Printf("Gossip: node left cluster: %s (%s:%d)", node.Name, node.Addr, node.Port)
}

// NotifyUpdate is called when a node's metadata is updated.
func (e *synapseEventDelegate) NotifyUpdate(node *memberlist.Node) {
	log.Printf("Gossip: node updated: %s (%s:%d)", node.Name, node.Addr, node.Port)
}
