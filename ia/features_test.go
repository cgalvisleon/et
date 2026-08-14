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

func TestExtractFeaturesEmptyStatement(t *testing.T) {
	f := ExtractFeatures("   ", nil, nil)
	if f.WordCount != 0 {
		t.Fatalf("expected zero word count for empty statement")
	}
}
