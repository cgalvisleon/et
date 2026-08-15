package ia

import "testing"

func TestExtractFeaturesVectorLength(t *testing.T) {
	f := ExtractFeatures("Yo estuve en la casa el martes 5 a las 10", nil, nil)
	v := f.Vector()
	if len(v) != FeatureCount {
		t.Fatalf("expected %d features, got %d", FeatureCount, len(v))
	}
	if f.WordCount == 0 {
		t.Fatalf("expected non-zero word count")
	}
	if f.DetailCount < 2 {
		t.Fatalf("expected at least 2 numeric details, got %v", f.DetailCount)
	}
}

func TestExtractFeaturesContradictsKB(t *testing.T) {
	kb := NewKnowledgeBase("kb-1")
	if _, err := kb.AddFact("Yo estuve en la casa el martes", 1, nil); err != nil {
		t.Fatalf("AddFact: %v", err)
	}

	f := ExtractFeatures("Yo no estuve en la casa el martes", kb, nil)
	if f.MaxKBSimilarity <= 0 {
		t.Fatalf("expected a similar existing fact to be found")
	}
	if f.ContradictsKB != 1 {
		t.Fatalf("expected negated restatement to be flagged as contradicting the KB")
	}
}

func TestExtractFeaturesContradictsKBViaTriples(t *testing.T) {
	kb := NewKnowledgeBase("kb-2")
	if _, err := kb.AddFact("Pague 100 dolares por el servicio.", 1, nil); err != nil {
		t.Fatalf("AddFact: %v", err)
	}

	// No negation word on either side, but same subject+predicate ("yo pague") with
	// a different object (100 vs 200) — the old negation-only heuristic would miss
	// this entirely.
	f := ExtractFeatures("Pague 200 dolares por el servicio.", kb, nil)
	if f.ContradictsKB != 1 {
		t.Fatalf("expected the differing amount to be flagged as a triple-level contradiction")
	}
}

func TestExtractFeaturesEmptyStatement(t *testing.T) {
	f := ExtractFeatures("   ", nil, nil)
	if f.WordCount != 0 {
		t.Fatalf("expected zero word count for empty statement")
	}
}
