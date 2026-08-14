package ia

import "errors"

// Verdict: the result of classifying a statement against a knowledge base.
// @param Statement string, IsTruth bool, Confidence float64, ContradictsFactID string
type Verdict struct {
	Statement         string  `json:"statement"`
	IsTruth           bool    `json:"is_truth"`
	Confidence        float64 `json:"confidence"`
	ContradictsFactID string  `json:"contradicts_fact_id,omitempty"`
}

// Classifier combines a trained Model with a Lexicon to turn a statement — optionally
// scored against a KnowledgeBase — into a Verdict.
// @param model *Model, lex *Lexicon
type Classifier struct {
	model *Model
	lex   *Lexicon
}

/**
* NewClassifier: builds a Classifier around model (lex nil falls back to
* DefaultLexiconES). Fails if model is nil.
* @param model *Model, lex *Lexicon
* @return *Classifier, error
**/
func NewClassifier(model *Model, lex *Lexicon) (*Classifier, error) {
	if model == nil {
		return nil, errors.New(MSG_MODEL_NOT_TRAINED)
	}
	if lex == nil {
		lex = DefaultLexiconES()
	}

	return &Classifier{model: model, lex: lex}, nil
}

/**
* Classify: scores statement against kb (nil for no knowledge-base context) and
* returns a Verdict. When the closest known fact was found to contradict statement,
* ContradictsFactID names it.
* @param kb *KnowledgeBase, statement string
* @return Verdict, error
**/
func (s *Classifier) Classify(kb *KnowledgeBase, statement string) (Verdict, error) {
	features := ExtractFeatures(statement, kb, s.lex)
	prob, err := s.model.Predict(features.Vector())
	if err != nil {
		return Verdict{}, err
	}

	verdict := Verdict{
		Statement:  statement,
		IsTruth:    prob >= 0.5,
		Confidence: confidenceFrom(prob),
	}

	if kb != nil && features.ContradictsKB == 1 {
		if fact, _ := ClosestFact(kb, statement); fact != nil {
			verdict.ContradictsFactID = fact.ID
		}
	}

	return verdict, nil
}

/**
* confidenceFrom: converts a truth probability into a confidence in [0.5, 1] for the
* winning side of the verdict (the probability itself when >= 0.5, otherwise its
* complement).
* @param prob float64
* @return float64
**/
func confidenceFrom(prob float64) float64 {
	if prob >= 0.5 {
		return prob
	}

	return 1 - prob
}
