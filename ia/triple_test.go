package ia

import "testing"

func TestExtractTriplesSingleClause(t *testing.T) {
	triples := extractTriples("Yo estuve en la oficina el lunes 3 de marzo a las 9 de la manana.")
	if len(triples) != 1 {
		t.Fatalf("expected 1 triple, got %d: %+v", len(triples), triples)
	}

	tr := triples[0]
	if tr.Subject != "yo" {
		t.Errorf("expected subject %q, got %q", "yo", tr.Subject)
	}
	if tr.Predicate != "estuve" {
		t.Errorf("expected predicate %q, got %q", "estuve", tr.Predicate)
	}
	if tr.Object == "" || tr.Object[:7] != "oficina" {
		t.Errorf("expected object to start with %q, got %q", "oficina", tr.Object)
	}
}

func TestExtractTriplesMultipleClauses(t *testing.T) {
	triples := extractTriples("Llegue a casa a las 8 y cene con mi familia.")
	if len(triples) != 2 {
		t.Fatalf("expected 2 triples, got %d: %+v", len(triples), triples)
	}

	if triples[0].Predicate != "llegue" || triples[0].Subject != "yo" {
		t.Errorf("unexpected first triple: %+v", triples[0])
	}
	if triples[1].Predicate != "cene" || triples[1].Object != "mi familia" {
		t.Errorf("unexpected second triple: %+v", triples[1])
	}
}

func TestExtractTriplesNonFirstPersonSubject(t *testing.T) {
	triples := extractTriples("El cielo es azul.")
	if len(triples) != 1 {
		t.Fatalf("expected 1 triple, got %d: %+v", len(triples), triples)
	}
	if triples[0].Subject != "cielo" {
		t.Errorf("expected subject %q, got %q", "cielo", triples[0].Subject)
	}
	if triples[0].Object != "azul" {
		t.Errorf("expected object %q, got %q", "azul", triples[0].Object)
	}
}

func TestExtractTriplesNoRecognizableVerb(t *testing.T) {
	triples := extractTriples("Mesa silla ventana puerta")
	if len(triples) != 0 {
		t.Errorf("expected no triples when no known verb is present, got %+v", triples)
	}
}

func TestAddFactPopulatesTriples(t *testing.T) {
	kb := NewKnowledgeBase("kb-triples")
	fact, err := kb.AddFact("Compre el carro el 12 de abril.", 1, nil)
	if err != nil {
		t.Fatalf("AddFact: %v", err)
	}
	if len(fact.Triples) != 1 || fact.Triples[0].Predicate != "compre" {
		t.Errorf("expected AddFact to populate Triples, got %+v", fact.Triples)
	}
}
