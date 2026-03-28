package ml

import (
	"math"
	"math/rand"
)

// IsolationForest implements the Isolation Forest anomaly detection algorithm.
// Points that are isolated quickly (short average path length) are anomalies.
type IsolationForest struct {
	trees     []*iTree
	numTrees  int
	maxDepth  int
	threshold float64 // anomaly score threshold (0-1)
}

// NewIsolationForest creates an untrained forest.
func NewIsolationForest(numTrees int, maxDepth int, threshold float64) *IsolationForest {
	if numTrees == 0 {
		numTrees = 100
	}
	if maxDepth == 0 {
		maxDepth = 10
	}
	if threshold == 0 {
		threshold = 0.6
	}
	return &IsolationForest{
		numTrees:  numTrees,
		maxDepth:  maxDepth,
		threshold: threshold,
	}
}

// Train builds the isolation forest from training data.
// Each row in data is a feature vector.
func (f *IsolationForest) Train(data [][]float64) {
	f.trees = make([]*iTree, f.numTrees)
	sampleSize := int(math.Min(256, float64(len(data))))

	for i := 0; i < f.numTrees; i++ {
		sample := subsample(data, sampleSize)
		f.trees[i] = buildITree(sample, 0, f.maxDepth)
	}
}

// Score returns the anomaly score for a point (0 = normal, 1 = anomalous).
func (f *IsolationForest) Score(point []float64) float64 {
	if len(f.trees) == 0 {
		return 0
	}

	var totalPath float64
	for _, tree := range f.trees {
		totalPath += float64(pathLength(point, tree, 0))
	}
	avgPath := totalPath / float64(len(f.trees))

	// Normalize using the average path length of unsuccessful search in BST
	n := float64(256) // expected sample size
	c := averagePathLength(n)
	score := math.Pow(2, -avgPath/c)
	return score
}

// IsAnomaly returns true if the point exceeds the anomaly threshold.
func (f *IsolationForest) IsAnomaly(point []float64) bool {
	return f.Score(point) > f.threshold
}

// Trained returns true if the forest has been trained.
func (f *IsolationForest) Trained() bool {
	return len(f.trees) > 0
}

// iTree is a single isolation tree node.
type iTree struct {
	left      *iTree
	right     *iTree
	splitAttr int     // feature index to split on
	splitVal  float64 // split value
	size      int     // number of samples at this node (for leaf nodes)
	isLeaf    bool
}

func buildITree(data [][]float64, depth, maxDepth int) *iTree {
	if len(data) <= 1 || depth >= maxDepth {
		return &iTree{isLeaf: true, size: len(data)}
	}

	dims := len(data[0])
	if dims == 0 {
		return &iTree{isLeaf: true, size: len(data)}
	}

	// Pick random attribute and split value
	attr := rand.Intn(dims)
	minVal, maxVal := minMax(data, attr)
	if minVal == maxVal {
		return &iTree{isLeaf: true, size: len(data)}
	}

	splitVal := minVal + rand.Float64()*(maxVal-minVal)

	var left, right [][]float64
	for _, row := range data {
		if row[attr] < splitVal {
			left = append(left, row)
		} else {
			right = append(right, row)
		}
	}

	return &iTree{
		splitAttr: attr,
		splitVal:  splitVal,
		left:      buildITree(left, depth+1, maxDepth),
		right:     buildITree(right, depth+1, maxDepth),
	}
}

func pathLength(point []float64, tree *iTree, depth int) int {
	if tree.isLeaf {
		return depth + int(averagePathLength(float64(tree.size)))
	}
	if point[tree.splitAttr] < tree.splitVal {
		return pathLength(point, tree.left, depth+1)
	}
	return pathLength(point, tree.right, depth+1)
}

// averagePathLength computes the average path of unsuccessful search in a BST.
func averagePathLength(n float64) float64 {
	if n <= 1 {
		return 0
	}
	return 2*(math.Log(n-1)+0.5772156649) - 2*(n-1)/n
}

func subsample(data [][]float64, size int) [][]float64 {
	if len(data) <= size {
		return data
	}
	perm := rand.Perm(len(data))
	sample := make([][]float64, size)
	for i := 0; i < size; i++ {
		sample[i] = data[perm[i]]
	}
	return sample
}

func minMax(data [][]float64, attr int) (float64, float64) {
	min := data[0][attr]
	max := data[0][attr]
	for _, row := range data[1:] {
		if row[attr] < min {
			min = row[attr]
		}
		if row[attr] > max {
			max = row[attr]
		}
	}
	return min, max
}
