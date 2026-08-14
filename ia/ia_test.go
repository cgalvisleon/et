package ia

import "testing"

func trainTestModel(t *testing.T) *Model {
	t.Helper()

	model, _, err := TrainFromCSV("testdata/sample_dataset.csv", nil, 300, 0.3, 0.75, 7)
	if err != nil {
		t.Fatalf("TrainFromCSV: %v", err)
	}

	return model
}

func TestEngineLearnAndVerify(t *testing.T) {
	model := trainTestModel(t)
	classifier, err := NewClassifier(model, nil)
	if err != nil {
		t.Fatalf("NewClassifier: %v", err)
	}

	engine, err := New(nil, classifier, 0)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	fact, err := engine.Learn("conv-1", "Yo estuve en la oficina el lunes 3 de marzo a las 9 de la manana.", 1, nil)
	if err != nil {
		t.Fatalf("Learn: %v", err)
	}
	if fact.Version != 1 {
		t.Fatalf("expected a first version fact, got version %d", fact.Version)
	}

	revised, err := engine.Revise("conv-1", fact.ID, "Yo estuve en la oficina el martes 4 de marzo a las 9 de la manana.", 0.8, nil)
	if err != nil {
		t.Fatalf("Revise: %v", err)
	}
	if revised.SupersedesID != fact.ID {
		t.Fatalf("expected revised fact to supersede %s, got %s", fact.ID, revised.SupersedesID)
	}

	verdict, err := engine.Verify("conv-1", "Compre el carro en la agencia el 12 de abril por 15000 dolares.")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verdict.Confidence <= 0 {
		t.Fatalf("expected a positive confidence, got %v", verdict.Confidence)
	}

	if !engine.IsLoaded("conv-1") {
		t.Fatalf("expected conv-1 to be loaded after Learn/Verify")
	}

	engine.Unload("conv-1")
	if engine.IsLoaded("conv-1") {
		t.Fatalf("expected conv-1 to be unloaded")
	}
}

func TestNewRequiresClassifier(t *testing.T) {
	if _, err := New(nil, nil, 0); err == nil {
		t.Fatalf("expected an error when classifier is nil")
	}
}
