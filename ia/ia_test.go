package ia

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeLLM is a test double for LLM: it returns a fixed answer/error and records the
// last prompt it was given, so tests can drive Engine.Ask's LLM branch without a
// real Ollama server.
type fakeLLM struct {
	answer     string
	err        error
	lastPrompt string
}

func (s *fakeLLM) Complete(ctx context.Context, prompt string) (string, error) {
	s.lastPrompt = prompt
	return s.answer, s.err
}

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

func newTestEngine(t *testing.T) *Engine {
	t.Helper()

	model := trainTestModel(t)
	classifier, err := NewClassifier(model, nil)
	if err != nil {
		t.Fatalf("NewClassifier: %v", err)
	}

	engine, err := New(nil, classifier, 0)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return engine
}

func TestEngineAskFindsRelevantFact(t *testing.T) {
	engine := newTestEngine(t)

	if _, err := engine.Learn("conv-ask", "Yo estuve en la oficina el lunes 3 de marzo a las 9 de la manana.", 1, nil); err != nil {
		t.Fatalf("Learn: %v", err)
	}

	answer, err := engine.Ask("conv-ask", "Donde estuve el lunes 3 de marzo?")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !answer.Found || answer.Fact == nil {
		t.Fatalf("expected a matching fact, got %+v", answer)
	}

	prefixes := []string{
		"Según lo que sé, ",
		"Creo que esto responde tu pregunta: ",
		"No estoy del todo seguro, pero esto es lo más cercano que tengo: ",
	}
	matched := false
	for _, prefix := range prefixes {
		if strings.HasPrefix(answer.Answer, prefix) {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("expected Answer to start with a confidence-graded prefix, got %q", answer.Answer)
	}
}

func TestEngineAskUsesLLMWhenConfigured(t *testing.T) {
	engine := newTestEngine(t)
	llm := &fakeLLM{answer: "Según lo que sé, el cielo es azul."}
	engine.UseLLM(llm)

	if _, err := engine.Learn("conv-llm", "El cielo es azul.", 1, nil); err != nil {
		t.Fatalf("Learn: %v", err)
	}

	answer, err := engine.Ask("conv-llm", "de que color es el cielo")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if answer.Source != "llm" || answer.Answer != llm.answer {
		t.Fatalf("expected the LLM's answer with Source=llm, got %+v", answer)
	}
	if !strings.Contains(llm.lastPrompt, "El cielo es azul.") {
		t.Fatalf("expected the prompt to ground on the learned fact, got %q", llm.lastPrompt)
	}
}

func TestEngineAskFallsBackWhenLLMFails(t *testing.T) {
	engine := newTestEngine(t)
	engine.UseLLM(&fakeLLM{err: errors.New("ollama unreachable")})

	if _, err := engine.Learn("conv-llm-fail", "El cielo es azul.", 1, nil); err != nil {
		t.Fatalf("Learn: %v", err)
	}

	answer, err := engine.Ask("conv-llm-fail", "de que color es el cielo")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if answer.Source == "llm" {
		t.Fatalf("expected a heuristic fallback answer, got Source=llm: %+v", answer)
	}
	if !answer.Found || answer.Fact == nil {
		t.Fatalf("expected the heuristic pipeline to still find the fact, got %+v", answer)
	}
}

func TestEngineAskFallsBackWhenLLMReturnsEmpty(t *testing.T) {
	engine := newTestEngine(t)
	engine.UseLLM(&fakeLLM{answer: "   "})

	if _, err := engine.Learn("conv-llm-empty", "El cielo es azul.", 1, nil); err != nil {
		t.Fatalf("Learn: %v", err)
	}

	answer, err := engine.Ask("conv-llm-empty", "de que color es el cielo")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if answer.Source == "llm" {
		t.Fatalf("expected a heuristic fallback answer for an empty LLM response, got %+v", answer)
	}
}

func TestEngineAskGreeting(t *testing.T) {
	engine := newTestEngine(t)

	answer, err := engine.Ask("conv-greet", "hola")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !answer.Found || answer.Answer != greetingAnswer {
		t.Fatalf("expected the fixed greeting answer, got %+v", answer)
	}
	if answer.Fact != nil || answer.Related != nil {
		t.Fatalf("expected no fact/related facts for a greeting, got %+v", answer)
	}
}

func TestListFactsAnswerSingular(t *testing.T) {
	engine := newTestEngine(t)

	if _, err := engine.Learn("conv-singular", "El cielo es azul.", 1, nil); err != nil {
		t.Fatalf("Learn: %v", err)
	}

	answer, err := engine.Ask("conv-singular", "que sabes?")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if want := "Tengo 1 hecho registrado en esta conversación."; answer.Answer != want {
		t.Fatalf("expected singular phrasing %q, got %q", want, answer.Answer)
	}
}

func TestEngineAskListsKnownFacts(t *testing.T) {
	engine := newTestEngine(t)

	if _, err := engine.Learn("conv-list", "Yo estuve en la oficina el lunes 3 de marzo a las 9 de la manana.", 1, nil); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	if _, err := engine.Learn("conv-list", "Compre el carro en la agencia el 12 de abril por 15000 dolares.", 1, nil); err != nil {
		t.Fatalf("Learn: %v", err)
	}

	answer, err := engine.Ask("conv-list", "Que hechos tienes en la base de conocimiento?")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !answer.Found {
		t.Fatalf("expected Found=true when the KB has facts, got %+v", answer)
	}
	if answer.Fact != nil {
		t.Fatalf("expected no single winning fact for a list-facts answer, got %+v", answer.Fact)
	}
	if len(answer.Related) != 2 {
		t.Fatalf("expected 2 related facts, got %d", len(answer.Related))
	}
}

func TestEngineAskListsFactsWhenKBEmpty(t *testing.T) {
	engine := newTestEngine(t)

	answer, err := engine.Ask("conv-empty", "Que sabes?")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if answer.Found {
		t.Fatalf("expected Found=false for an empty knowledge base, got %+v", answer)
	}
}

func TestEngineFacts(t *testing.T) {
	engine := newTestEngine(t)

	if _, err := engine.Learn("conv-facts", "Yo estuve en la oficina el lunes 3 de marzo a las 9 de la manana.", 1, nil); err != nil {
		t.Fatalf("Learn: %v", err)
	}
	if _, err := engine.Learn("conv-facts", "Compre el carro en la agencia el 12 de abril por 15000 dolares.", 1, nil); err != nil {
		t.Fatalf("Learn: %v", err)
	}

	facts, err := engine.Facts("conv-facts")
	if err != nil {
		t.Fatalf("Facts: %v", err)
	}
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(facts))
	}
}
