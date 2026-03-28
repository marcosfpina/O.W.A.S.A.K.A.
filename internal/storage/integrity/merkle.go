package integrity

import (
	"crypto/sha256"
	"encoding/hex"
)

// MerkleTree provides a hash tree over a set of data blocks.
// Used to verify integrity of audit log entries and snapshots.
type MerkleTree struct {
	Leaves [][]byte // leaf hashes
	Nodes  [][]byte // all tree nodes (bottom-up, left-to-right)
	Root   []byte   // root hash
}

// BuildTree constructs a Merkle tree from data blocks.
func BuildTree(blocks [][]byte) *MerkleTree {
	if len(blocks) == 0 {
		return &MerkleTree{}
	}

	// Hash all leaves
	leaves := make([][]byte, len(blocks))
	for i, b := range blocks {
		h := sha256.Sum256(b)
		leaves[i] = h[:]
	}

	// Pad to even number if necessary
	nodes := make([][]byte, len(leaves))
	copy(nodes, leaves)
	if len(nodes)%2 != 0 {
		nodes = append(nodes, nodes[len(nodes)-1])
	}

	allNodes := make([][]byte, 0, len(nodes)*2)
	allNodes = append(allNodes, nodes...)

	// Build tree bottom-up
	for len(nodes) > 1 {
		var level [][]byte
		for i := 0; i < len(nodes); i += 2 {
			combined := append(nodes[i], nodes[i+1]...)
			h := sha256.Sum256(combined)
			level = append(level, h[:])
		}
		allNodes = append(allNodes, level...)
		if len(level) > 1 && len(level)%2 != 0 {
			level = append(level, level[len(level)-1])
		}
		nodes = level
	}

	return &MerkleTree{
		Leaves: leaves,
		Nodes:  allNodes,
		Root:   nodes[0],
	}
}

// RootHex returns the root hash as a hex string.
func (t *MerkleTree) RootHex() string {
	if t.Root == nil {
		return ""
	}
	return hex.EncodeToString(t.Root)
}

// VerifyLeaf checks that a given data block exists at the specified index.
func (t *MerkleTree) VerifyLeaf(index int, data []byte) bool {
	if index < 0 || index >= len(t.Leaves) {
		return false
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]) == hex.EncodeToString(t.Leaves[index])
}

// HashBlock returns the SHA-256 hash of a data block.
func HashBlock(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
