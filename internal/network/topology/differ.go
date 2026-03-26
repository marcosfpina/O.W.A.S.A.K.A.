package topology

// ChangeKind classifies topology mutations
type ChangeKind string

const (
	ChangeNodeAdded   ChangeKind = "node_added"
	ChangeNodeRemoved ChangeKind = "node_removed"
	ChangeEdgeAdded   ChangeKind = "edge_added"
)

// Change describes a single topology mutation
type Change struct {
	Kind   ChangeKind `json:"kind"`
	NodeID string     `json:"node_id,omitempty"`
	Edge   *Edge      `json:"edge,omitempty"`
}

// Diff computes the delta between two consecutive snapshots
func Diff(prev, next GraphSnapshot) []Change {
	var changes []Change

	prevNodes := make(map[string]struct{}, len(prev.Nodes))
	for _, n := range prev.Nodes {
		prevNodes[n.ID] = struct{}{}
	}

	nextNodes := make(map[string]struct{}, len(next.Nodes))
	for _, n := range next.Nodes {
		nextNodes[n.ID] = struct{}{}
		if _, exists := prevNodes[n.ID]; !exists {
			changes = append(changes, Change{Kind: ChangeNodeAdded, NodeID: n.ID})
		}
	}

	for _, n := range prev.Nodes {
		if _, exists := nextNodes[n.ID]; !exists {
			changes = append(changes, Change{Kind: ChangeNodeRemoved, NodeID: n.ID})
		}
	}

	prevEdges := make(map[string]struct{}, len(prev.Edges))
	for _, e := range prev.Edges {
		prevEdges[e.Source+"->"+e.Target] = struct{}{}
	}
	for i := range next.Edges {
		e := next.Edges[i]
		if _, exists := prevEdges[e.Source+"->"+e.Target]; !exists {
			changes = append(changes, Change{Kind: ChangeEdgeAdded, Edge: &e})
		}
	}

	return changes
}
