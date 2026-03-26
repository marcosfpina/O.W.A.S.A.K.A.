package topology

import (
	"github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"
	"github.com/marcosfpina/O.W.A.S.A.K.A/pkg/logging"
)

// ChangeCallback is invoked whenever the topology mutates
type ChangeCallback func(snap GraphSnapshot)

// Builder maintains the topology graph from live asset and event streams
type Builder struct {
	graph    *Graph
	logger   *logging.Logger
	onChange ChangeCallback
}

// NewBuilder creates a topology builder backed by a fresh graph
func NewBuilder(logger *logging.Logger) *Builder {
	return &Builder{
		graph:  NewGraph(),
		logger: logger,
	}
}

// OnChange registers a callback fired on every topology mutation
func (b *Builder) OnChange(fn ChangeCallback) {
	b.onChange = fn
}

// OnAsset integrates a discovered asset into the topology graph
func (b *Builder) OnAsset(a models.Asset) {
	label := a.Hostname
	if label == "" {
		label = a.MAC
	}
	if label == "" {
		label = a.IP
	}

	changed := b.graph.UpsertNode(Node{
		ID:       a.IP,
		Label:    label,
		Type:     NodeTypeHost,
		MAC:      a.MAC,
		OS:       a.OS,
		Ports:    a.Ports,
		LastSeen: a.LastSeen,
	})

	if changed {
		b.logger.Infow("Topology: node upserted", "ip", a.IP, "label", label, "total_nodes", b.graph.NodeCount())
		if b.onChange != nil {
			b.onChange(b.graph.Snapshot())
		}
	}
}

// OnEvent extracts source/destination from a network event to build edges
func (b *Builder) OnEvent(e models.NetworkEvent) {
	if e.Source == "" || e.Destination == "" {
		return
	}

	// Ensure both endpoints exist as nodes (may be upgraded later by OnAsset)
	b.graph.UpsertNode(Node{ID: e.Source, Label: e.Source, Type: NodeTypeUnknown, LastSeen: e.Timestamp})
	b.graph.UpsertNode(Node{ID: e.Destination, Label: e.Destination, Type: NodeTypeUnknown, LastSeen: e.Timestamp})

	isNew := b.graph.UpsertEdge(e.Source, e.Destination, string(e.Type))
	if isNew && b.onChange != nil {
		b.onChange(b.graph.Snapshot())
	}
}

// Snapshot returns the current topology
func (b *Builder) Snapshot() GraphSnapshot {
	return b.graph.Snapshot()
}
