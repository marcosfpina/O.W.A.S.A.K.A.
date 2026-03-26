package topology

import "encoding/json"

// D3Node is the node shape expected by the Svelte D3 force graph
type D3Node struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Type  NodeType `json:"type"`
	OS    string   `json:"os,omitempty"`
	Ports []int    `json:"ports,omitempty"`
}

// D3Link is the edge shape expected by the Svelte D3 force graph
type D3Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Count  int    `json:"count"`
}

// D3Graph is the full topology payload for the frontend
type D3Graph struct {
	Nodes []D3Node `json:"nodes"`
	Links []D3Link `json:"links"`
}

// ToD3 converts a GraphSnapshot to the D3-compatible format
func ToD3(snap GraphSnapshot) D3Graph {
	g := D3Graph{
		Nodes: make([]D3Node, 0, len(snap.Nodes)),
		Links: make([]D3Link, 0, len(snap.Edges)),
	}
	for _, n := range snap.Nodes {
		g.Nodes = append(g.Nodes, D3Node{
			ID:    n.ID,
			Label: n.Label,
			Type:  n.Type,
			OS:    n.OS,
			Ports: n.Ports,
		})
	}
	for _, e := range snap.Edges {
		g.Links = append(g.Links, D3Link{
			Source: e.Source,
			Target: e.Target,
			Count:  e.Count,
		})
	}
	return g
}

// MarshalD3 returns JSON bytes of the D3-compatible topology
func MarshalD3(snap GraphSnapshot) ([]byte, error) {
	return json.Marshal(ToD3(snap))
}
