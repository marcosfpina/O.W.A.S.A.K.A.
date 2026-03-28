package ml

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestIsolationForest_TrainAndScore(t *testing.T) {
	forest := NewIsolationForest(50, 8, 0.6)

	if forest.Trained() {
		t.Fatal("forest should not be trained before Train()")
	}

	// Generate normal data cluster around (1, 1, 1, 1)
	var data [][]float64
	for i := 0; i < 100; i++ {
		data = append(data, []float64{
			1 + rand.Float64()*0.5,
			1 + rand.Float64()*0.5,
			1 + rand.Float64()*0.5,
			1 + rand.Float64()*0.5,
		})
	}

	forest.Train(data)

	if !forest.Trained() {
		t.Fatal("forest should be trained after Train()")
	}

	// Normal point — should have LOW anomaly score
	normalScore := forest.Score([]float64{1.2, 1.1, 1.3, 1.0})
	if normalScore > 0.7 {
		t.Fatalf("normal point scored too high: %f", normalScore)
	}

	// Anomalous point — far from cluster
	anomalyScore := forest.Score([]float64{100, 100, 100, 100})
	if anomalyScore < 0.5 {
		t.Fatalf("anomaly point scored too low: %f", anomalyScore)
	}
}

func TestIsolationForest_IsAnomaly(t *testing.T) {
	forest := NewIsolationForest(50, 8, 0.6)

	var data [][]float64
	for i := 0; i < 100; i++ {
		data = append(data, []float64{rand.Float64(), rand.Float64()})
	}
	forest.Train(data)

	// Extreme outlier should be anomalous
	if !forest.IsAnomaly([]float64{1000, 1000}) {
		t.Fatal("extreme outlier should be flagged as anomaly")
	}
}

func TestIsolationForest_UntrainedReturnsZero(t *testing.T) {
	forest := NewIsolationForest(10, 5, 0.6)
	score := forest.Score([]float64{1, 2, 3})
	if score != 0 {
		t.Fatalf("untrained forest should return 0, got %f", score)
	}
}

func TestIsolationForest_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.gob")

	// Train and save
	forest := NewIsolationForest(20, 5, 0.6)
	var data [][]float64
	for i := 0; i < 50; i++ {
		data = append(data, []float64{rand.Float64(), rand.Float64()})
	}
	forest.Train(data)

	originalScore := forest.Score([]float64{0.5, 0.5})

	if err := forest.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("model file not created: %v", err)
	}

	// Load into new forest
	loaded := NewIsolationForest(20, 5, 0.6)
	if err := loaded.Load(path); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !loaded.Trained() {
		t.Fatal("loaded forest should be trained")
	}

	loadedScore := loaded.Score([]float64{0.5, 0.5})

	// Scores should be identical (same trees)
	if originalScore != loadedScore {
		t.Fatalf("scores differ after load: %f vs %f", originalScore, loadedScore)
	}
}

func TestIsolationForest_LoadNonExistent(t *testing.T) {
	forest := NewIsolationForest(10, 5, 0.6)
	err := forest.Load("/nonexistent/path.gob")
	if err == nil {
		t.Fatal("expected error loading non-existent file")
	}
}
