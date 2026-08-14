package ia

import (
	"strings"
	"testing"
)

func TestReadCSVDatasetParsesLabelsAndSkipsHeader(t *testing.T) {
	csv := "text,label\n\"hola mundo\",1\n\"otro texto\",0\n"
	examples, err := ReadCSVDataset(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("ReadCSVDataset: %v", err)
	}
	if len(examples) != 2 {
		t.Fatalf("expected 2 examples, got %d", len(examples))
	}
	if !examples[0].Label || examples[1].Label {
		t.Fatalf("unexpected labels: %+v", examples)
	}
}

func TestReadCSVDatasetInvalidLabel(t *testing.T) {
	csv := "text,label\n\"hola\",maybe\n"
	if _, err := ReadCSVDataset(strings.NewReader(csv)); err == nil {
		t.Fatalf("expected an error for an invalid label")
	}
}

func TestSplitDatasetRatio(t *testing.T) {
	examples := make([]TrainingExample, 10)
	for i := range examples {
		examples[i] = TrainingExample{Text: "x", Label: i%2 == 0}
	}

	train, test := SplitDataset(examples, 0.8, 42)
	if len(train) != 8 || len(test) != 2 {
		t.Fatalf("expected an 8/2 split, got %d/%d", len(train), len(test))
	}
}

func TestTrainFromCSVPipeline(t *testing.T) {
	model, metrics, err := TrainFromCSV("testdata/sample_dataset.csv", nil, 300, 0.3, 0.75, 7)
	if err != nil {
		t.Fatalf("TrainFromCSV: %v", err)
	}
	if len(model.Weights) != FeatureCount {
		t.Fatalf("expected %d weights, got %d", FeatureCount, len(model.Weights))
	}
	if metrics.Accuracy < 0.5 {
		t.Errorf("expected the sample dataset to be at least as good as chance, got accuracy %v", metrics.Accuracy)
	}
}
