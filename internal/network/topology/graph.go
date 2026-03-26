package topology

import (
	"sync"
	"time"
)

// NodeType classifies network participants
type NodeType string

const (
	NodeTypeHost      NodeType = "host"
	NodeTypeRouter    NodeType = "router"
	NodeTypeContainer NodeType = "container"
	NodeTypeVM        NodeType = "vm"
	NodeTypeUnknown   NodeType = "unknown"
)

// Node represents a network participant keyed by IP
type Node struct {
	ID       string    `json:"id"`
	Label    string    `json:"label"`
	Type     NodeType  `json:"type"`
	MAC      string    `json:"mac,omitempty"`
	OS       string    `json:"os,omitempty"`
	Ports    []int     `json:"ports,omitempty"`
	LastSeen time.Time `json:"last_seen"`
}

// Edge represents an observed communication path between two IPs
type Edge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Count    int    `json:"count"`
	Protocol string `json:"protocol,omitempty"`
}

// GraphSnapshot is a point-in-time serializable copy of the topology
type GraphSnapshot struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Graph is the authoritative in-memory network topology store
type Graph struct {
	mu    sync.RWMutex
	nodes map[string]*Node
	edges map[string]*Edge // key: "src->dst"
}

// NewGraph initializes an empty topology graph
func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]*Node),
		edges: make(map[string]*Edge),
	}
}

// UpsertNode adds or updates a node. Returns true if topology changed.
func (g *Graph) UpsertNode(n Node) (changed bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	existing, ok := g.nodes[n.ID]
	if !ok {
		g.nodes[n.ID] = &n
		return true
	}
	if n.Label != "" && n.Label != existing.Label {
		existing.Label = n.Label
		changed = true
	}
	if n.MAC != "" && n.MAC != existing.MAC {
		existing.MAC = n.MAC
		changed = true
	}
	if n.OS != "" && n.OS != existing.OS {
		existing.OS = n.OS
		changed = true
	}
	if len(n.Ports) > 0 {
		existing.Ports = n.Ports
		changed = true
	}
	if n.Type != NodeTypeUnknown && n.Type != existing.Type {
		existing.Type = n.Type
		changed = true
	}
	existing.LastSeen = n.LastSeen
	return changed
}

// UpsertEdge records or increments a connection. Returns true if edge is new.
func (g *Graph) UpsertEdge(src, dst, protocol string) (isNew bool) {
	if src == dst || src == "" || dst == "" {
		return false
	}
	key := src + "->" + dst
	g.mu.Lock()
	defer g.mu.Unlock()
	if e, ok := g.edges[key]; ok {
		e.Count++
		return false
	}
	g.edges[key] = &Edge{Source: src, Target: dst, Count: 1, Protocol: protocol}
	return true
}

// NodeCount returns the number of nodes in the graph
func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// Snapshot returns a serializable copy of the current topology
func (g *Graph) Snapshot() GraphSnapshot {
	g.mu.RLock()
	defer g.mu.RUnlock()
	snap := GraphSnapshot{
		Nodes: make([]Node, 0, len(g.nodes)),
		Edges: make([]Edge, 0, len(g.edges)),
	}
	for _, n := range g.nodes {
		snap.Nodes = append(snap.Nodes, *n)
	}
	for _, e := range g.edges {
		snap.Edges = append(snap.Edges, *e)
	}
	return snap
}
