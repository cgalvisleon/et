package ia

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
)

// Model: a binary logistic regression classifier over a fixed-length feature vector.
// Predict(x) close to 1 means the statement looks true, close to 0 means it looks
// like a lie.
// @param Weights []float64, Bias float64
type Model struct {
	Weights []float64 `json:"weights"`
	Bias    float64   `json:"bias"`
}

/**
* NewModel: returns a Model with zero-valued weights for featureCount inputs, ready
* to be fit via Train.
* @param featureCount int
* @return *Model
**/
func NewModel(featureCount int) *Model {
	return &Model{Weights: make([]float64, featureCount)}
}

/**
* sigmoid: the logistic function, squashing z into (0,1).
* @param z float64
* @return float64
**/
func sigmoid(z float64) float64 {
	return 1 / (1 + math.Exp(-z))
}

/**
* Predict: returns the probability, in [0,1], that features corresponds to a true
* statement. Fails if the model has no trained weights or features has the wrong
* length.
* @param features []float64
* @return float64, error
**/
func (s *Model) Predict(features []float64) (float64, error) {
	if len(s.Weights) == 0 {
		return 0, errors.New(MSG_MODEL_NOT_TRAINED)
	}
	if len(features) != len(s.Weights) {
		return 0, fmt.Errorf("expected %d features, got %d", len(s.Weights), len(features))
	}

	z := s.Bias
	for i, w := range s.Weights {
		z += w * features[i]
	}

	return sigmoid(z), nil
}

/**
* Train: fits the model's weights/bias to X (one feature vector per row) and y
* (matching 0/1 labels, 1 = true statement) using batch gradient descent on the
* logistic loss, for the given number of epochs and learning rate lr. Feature scales
* are not normalized internally — callers should keep feature magnitudes comparable
* (see Features.Vector).
* @param X [][]float64, y []float64, epochs int, lr float64
* @return error
**/
func (s *Model) Train(X [][]float64, y []float64, epochs int, lr float64) error {
	if len(X) == 0 || len(X) != len(y) {
		return errors.New("X and y must be non-empty and the same length")
	}

	featureCount := len(X[0])
	for _, row := range X {
		if len(row) != featureCount {
			return errors.New("all feature vectors must have the same length")
		}
	}

	if len(s.Weights) != featureCount {
		s.Weights = make([]float64, featureCount)
	}

	n := float64(len(X))
	for range epochs {
		gradW := make([]float64, featureCount)
		gradB := 0.0

		for i, row := range X {
			z := s.Bias
			for j, w := range s.Weights {
				z += w * row[j]
			}
			pred := sigmoid(z)
			diff := pred - y[i]

			for j := range gradW {
				gradW[j] += diff * row[j]
			}
			gradB += diff
		}

		for j := range s.Weights {
			s.Weights[j] -= lr * gradW[j] / n
		}
		s.Bias -= lr * gradB / n
	}

	return nil
}

/**
* Save: serializes the model's weights/bias as JSON to path.
* @param path string
* @return error
**/
func (s *Model) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

/**
* LoadModel: reads a Model previously written by Save.
* @param path string
* @return *Model, error
**/
func LoadModel(path string) (*Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	model := &Model{}
	if err := json.Unmarshal(data, model); err != nil {
		return nil, err
	}

	return model, nil
}
