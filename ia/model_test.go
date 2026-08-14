package ia

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModelTrainAndPredict(t *testing.T) {
	// Two clusters, linearly separable along the first feature.
	X := [][]float64{
		{5, 1}, {6, 1}, {7, 0}, {8, 1},
		{-5, 1}, {-6, 0}, {-7, 1}, {-8, 0},
	}
	y := []float64{1, 1, 1, 1, 0, 0, 0, 0}

	model := NewModel(2)
	if err := model.Train(X, y, 500, 0.1); err != nil {
		t.Fatalf("Train: %v", err)
	}

	p, err := model.Predict([]float64{6.5, 1})
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if p < 0.7 {
		t.Errorf("expected high probability for a positive-cluster point, got %v", p)
	}

	p, err = model.Predict([]float64{-6.5, 1})
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if p > 0.3 {
		t.Errorf("expected low probability for a negative-cluster point, got %v", p)
	}
}

func TestModelSaveLoad(t *testing.T) {
	model := NewModel(2)
	if err := model.Train([][]float64{{1, 0}, {0, 1}}, []float64{1, 0}, 10, 0.1); err != nil {
		t.Fatalf("Train: %v", err)
	}

	path := filepath.Join(t.TempDir(), "model.json")
	if err := model.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadModel(path)
	if err != nil {
		t.Fatalf("LoadModel: %v", err)
	}
	if len(loaded.Weights) != len(model.Weights) {
		t.Fatalf("expected %d weights, got %d", len(model.Weights), len(loaded.Weights))
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected model file to exist: %v", err)
	}
}

func TestModelPredictUntrained(t *testing.T) {
	model := &Model{}
	if _, err := model.Predict([]float64{1, 2}); err == nil {
		t.Fatalf("expected an error predicting with an untrained model")
	}
}
