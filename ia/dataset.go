package ia

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"

	_ "embed"
)

// TrainingExample: a single labeled statement used to fit or evaluate a Model.
// @param Text string, Label bool
type TrainingExample struct {
	Text  string
	Label bool
}

//go:embed testdata/sample_dataset.csv
var sampleDatasetCSV []byte

/**
* SampleDataset: returns the small, hand-crafted Spanish dataset bundled with this
* package (see testdata/sample_dataset.csv). It exists so the training pipeline works
* out of the box with no external files; replace it with a real deception-detection
* dataset (e.g. Perez-Rosas et al. or Ott et al.) for production-quality accuracy.
* @return []TrainingExample, error
**/
func SampleDataset() ([]TrainingExample, error) {
	return ReadCSVDataset(bytes.NewReader(sampleDatasetCSV))
}

/**
* LoadCSVDataset: reads a two-column (text,label) CSV file from path. See
* ReadCSVDataset for the accepted label values and header handling.
* @param path string
* @return []TrainingExample, error
**/
func LoadCSVDataset(path string) ([]TrainingExample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ReadCSVDataset(f)
}

/**
* ReadCSVDataset: parses a two-column (text,label) CSV from r. label accepts
* "1"/"0", "true"/"false", "verdad"/"mentira" and "truthful"/"deceptive"
* (case-insensitive). A header row whose label column does not parse is skipped.
* @param r io.Reader
* @return []TrainingExample, error
**/
func ReadCSVDataset(r io.Reader) ([]TrainingExample, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = 2

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	result := make([]TrainingExample, 0, len(rows))
	for i, row := range rows {
		label, err := parseLabel(row[1])
		if err != nil {
			if i == 0 {
				continue
			}
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		result = append(result, TrainingExample{Text: row[0], Label: label})
	}

	if len(result) == 0 {
		return nil, errors.New("dataset is empty")
	}

	return result, nil
}

/**
* parseLabel: converts a raw CSV label cell into a bool, or an error if unrecognized.
* @param raw string
* @return bool, error
**/
func parseLabel(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "verdad", "truthful":
		return true, nil
	case "0", "false", "mentira", "deceptive":
		return false, nil
	}

	return false, fmt.Errorf("invalid label %q", raw)
}

/**
* SplitDataset: deterministically shuffles examples (seed) and splits them into
* train/test sets according to trainRatio (e.g. 0.8 for an 80/20 split).
* @param examples []TrainingExample, trainRatio float64, seed int64
* @return train, test []TrainingExample
**/
func SplitDataset(examples []TrainingExample, trainRatio float64, seed int64) (train, test []TrainingExample) {
	shuffled := make([]TrainingExample, len(examples))
	copy(shuffled, examples)

	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	cut := int(float64(len(shuffled)) * trainRatio)
	return shuffled[:cut], shuffled[cut:]
}

// Metrics: evaluation results of a trained Model against a labeled test set.
// @param Accuracy, Precision, Recall, F1 float64
type Metrics struct {
	Accuracy  float64
	Precision float64
	Recall    float64
	F1        float64
}

/**
* Evaluate: scores model's predictions (threshold 0.5) against a labeled test set,
* extracting features with lex (nil falls back to DefaultLexiconES). No knowledge
* base context is used, since offline evaluation examples are not tied to any
* conversation.
* @param model *Model, test []TrainingExample, lex *Lexicon
* @return Metrics, error
**/
func Evaluate(model *Model, test []TrainingExample, lex *Lexicon) (Metrics, error) {
	var tp, tn, fp, fn float64
	for _, ex := range test {
		prob, err := model.Predict(ExtractFeatures(ex.Text, nil, lex).Vector())
		if err != nil {
			return Metrics{}, err
		}
		predicted := prob >= 0.5

		switch {
		case predicted && ex.Label:
			tp++
		case !predicted && !ex.Label:
			tn++
		case predicted && !ex.Label:
			fp++
		case !predicted && ex.Label:
			fn++
		}
	}

	metrics := Metrics{}
	if total := tp + tn + fp + fn; total > 0 {
		metrics.Accuracy = (tp + tn) / total
	}
	if tp+fp > 0 {
		metrics.Precision = tp / (tp + fp)
	}
	if tp+fn > 0 {
		metrics.Recall = tp / (tp + fn)
	}
	if metrics.Precision+metrics.Recall > 0 {
		metrics.F1 = 2 * metrics.Precision * metrics.Recall / (metrics.Precision + metrics.Recall)
	}

	return metrics, nil
}

/**
* TrainFromExamples: extracts features for every example, splits them into
* train/test (trainRatio, seed), fits a Model over the train split for epochs at
* learning rate lr, and evaluates it on the held-out split.
* @param examples []TrainingExample, lex *Lexicon, epochs int, lr, trainRatio float64,
* seed int64
* @return *Model, Metrics, error
**/
func TrainFromExamples(examples []TrainingExample, lex *Lexicon, epochs int, lr float64, trainRatio float64, seed int64) (*Model, Metrics, error) {
	if len(examples) < 4 {
		return nil, Metrics{}, errors.New("need at least 4 examples to train and evaluate")
	}

	train, test := SplitDataset(examples, trainRatio, seed)
	if len(train) == 0 || len(test) == 0 {
		return nil, Metrics{}, errors.New("train/test split produced an empty set")
	}

	X := make([][]float64, len(train))
	y := make([]float64, len(train))
	for i, ex := range train {
		X[i] = ExtractFeatures(ex.Text, nil, lex).Vector()
		y[i] = boolToFloat(ex.Label)
	}

	model := NewModel(FeatureCount)
	if err := model.Train(X, y, epochs, lr); err != nil {
		return nil, Metrics{}, err
	}

	metrics, err := Evaluate(model, test, lex)
	if err != nil {
		return nil, Metrics{}, err
	}

	return model, metrics, nil
}

/**
* TrainFromCSV: convenience wrapper combining LoadCSVDataset and TrainFromExamples.
* @param path string, lex *Lexicon, epochs int, lr, trainRatio float64, seed int64
* @return *Model, Metrics, error
**/
func TrainFromCSV(path string, lex *Lexicon, epochs int, lr float64, trainRatio float64, seed int64) (*Model, Metrics, error) {
	examples, err := LoadCSVDataset(path)
	if err != nil {
		return nil, Metrics{}, err
	}

	return TrainFromExamples(examples, lex, epochs, lr, trainRatio, seed)
}

/**
* boolToFloat: converts a label bool into the 0/1 float expected by Model.Train.
* @param b bool
* @return float64
**/
func boolToFloat(b bool) float64 {
	if b {
		return 1
	}

	return 0
}
