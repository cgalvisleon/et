package ia

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/cgalvisleon/et/et"
)

// Engine is the package's public entry point. It combines a Manager (in-memory
// knowledge bases with idle eviction, see manager.go) and a Classifier (trained
// model + linguistic features, see classifier.go) into a small, conversation-scoped
// API: learn new truths, revise them as more context arrives, and verify whether a
// new statement looks true or like a lie against what is already known.
// @param manager *Manager, classifier *Classifier
type Engine struct {
	manager    *Manager
	classifier *Classifier
}

/**
* New: builds an Engine backed by store (nil for in-memory-only knowledge bases) and
* classifier, evicting knowledge bases idle for more than idleTTL (DefaultIdleTTL
* when <= 0). Fails if classifier is nil.
* @param store Store, classifier *Classifier, idleTTL time.Duration
* @return *Engine, error
**/
func New(store Store, classifier *Classifier, idleTTL time.Duration) (*Engine, error) {
	if classifier == nil {
		return nil, errors.New(MSG_MODEL_NOT_TRAINED)
	}

	return &Engine{
		manager:    NewManager(store, idleTTL),
		classifier: classifier,
	}, nil
}

/**
* Learn: records statement as a new truth inside the kbId knowledge base (loading or
* creating it as needed) and returns the stored Fact.
* @param kbId, statement string, confidence float64, ctx et.Json
* @return *Fact, error
**/
func (s *Engine) Learn(kbId, statement string, confidence float64, ctx et.Json) (*Fact, error) {
	kb, err := s.manager.Load(kbId)
	if err != nil {
		return nil, err
	}

	return kb.AddFact(statement, confidence, ctx)
}

/**
* Revise: updates factId inside kbId with a new statement/confidence, superseding the
* previous version instead of discarding its history — used when more context changes
* what is known to be true.
* @param kbId, factId, statement string, confidence float64, ctx et.Json
* @return *Fact, error
**/
func (s *Engine) Revise(kbId, factId, statement string, confidence float64, ctx et.Json) (*Fact, error) {
	kb, err := s.manager.Load(kbId)
	if err != nil {
		return nil, err
	}

	return kb.UpdateFact(factId, statement, confidence, ctx)
}

/**
* Verify: classifies statement as truth or lie against kbId's current knowledge
* (loading or creating the knowledge base as needed).
* @param kbId, statement string
* @return Verdict, error
**/
func (s *Engine) Verify(kbId, statement string) (Verdict, error) {
	kb, err := s.manager.Load(kbId)
	if err != nil {
		return Verdict{}, err
	}

	return s.classifier.Classify(kb, statement)
}

// MinAskSimilarity is the default minimum StatementSimilarity score a fact needs to
// be considered a match for Engine.Ask.
const MinAskSimilarity = 0.2

// maxAskRelated bounds how many supporting facts Engine.Ask returns besides the best
// match.
const maxAskRelated = 4

// Confidence thresholds naturalAnswer uses to phrase Engine.Ask's answer. Framing
// the reply by match confidence (rather than trying to parse whether the question is
// yes/no or wh-, which is brittle in Spanish) keeps the wording honest and natural
// for either kind of question.
const (
	highConfidence   = 0.6
	mediumConfidence = 0.35
)

// greetingAnswer is the fixed reply Engine.Ask gives when isGreetingIntent matches.
const greetingAnswer = "¡Hola! Pregúntame algo y te responderé según lo que sepa en esta conversación."

// AskResult: the answer to a question asked against a KnowledgeBase's known facts.
// @param Question, Answer string, Fact *Fact, Score float64, Found bool,
// Related []*Fact
type AskResult struct {
	Question string  `json:"question"`
	Answer   string  `json:"answer"`
	Fact     *Fact   `json:"fact,omitempty"`
	Score    float64 `json:"score"`
	Found    bool    `json:"found"`
	Related  []*Fact `json:"related,omitempty"`
}

/**
* Ask: answers question using what kbId already knows (loading or creating the
* knowledge base as needed), instead of judging whether question itself is true or
* false — that is what Verify is for. It returns the closest matching fact's
* statement as Answer, plus any other relevant facts, or Found=false with a fallback
* message when nothing in the knowledge base is similar enough.
* @param kbId, question string
* @return AskResult, error
**/
func (s *Engine) Ask(kbId, question string) (AskResult, error) {
	kb, err := s.manager.Load(kbId)
	if err != nil {
		return AskResult{}, err
	}

	if isGreetingIntent(question) {
		return AskResult{Question: question, Answer: greetingAnswer, Found: true}, nil
	}

	if isListFactsIntent(question) {
		return listFactsAnswer(kb, question), nil
	}

	relevant := RelevantFacts(kb, question, maxAskRelated, MinAskSimilarity)
	if len(relevant) == 0 {
		return AskResult{
			Question: question,
			Answer:   "No tengo información sobre eso en esta base de conocimiento.",
			Found:    false,
		}, nil
	}

	best := relevant[0]
	score := StatementSimilarity(question, best.Statement)
	related := relevant[1:]

	answer := naturalAnswer(best, score)
	if len(related) > 0 {
		answer += fmt.Sprintf(" (también tengo %s sobre este tema)", pluralize(len(related), "hecho relacionado", "hechos relacionados"))
	}

	return AskResult{
		Question: question,
		Answer:   answer,
		Fact:     best,
		Score:    score,
		Found:    true,
		Related:  related,
	}, nil
}

/**
* naturalAnswer: phrases fact's statement as an answer, with a lead-in whose
* confidence matches score — honest about uncertainty instead of always stating the
* fact flatly.
* @param fact *Fact, score float64
* @return string
**/
func naturalAnswer(fact *Fact, score float64) string {
	switch {
	case score >= highConfidence:
		return "Según lo que sé, " + fact.Statement
	case score >= mediumConfidence:
		return "Creo que esto responde tu pregunta: " + fact.Statement
	default:
		return "No estoy del todo seguro, pero esto es lo más cercano que tengo: " + fact.Statement
	}
}

/**
* pluralize: returns singular or plural depending on n, prefixed with n itself (e.g.
* pluralize(1, "hecho registrado", "hechos registrados") -> "1 hecho registrado").
* @param n int, singular, plural string
* @return string
**/
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}

	return fmt.Sprintf("%d %s", n, plural)
}

/**
* listFactsAnswer: answers a "what do you know" meta-question with every active fact
* in kb, ordered from oldest to newest, instead of a single best match — used by Ask
* when isListFactsIntent detects that shape of question.
* @param kb *KnowledgeBase, question string
* @return AskResult
**/
func listFactsAnswer(kb *KnowledgeBase, question string) AskResult {
	facts := kb.ActiveFacts()
	if len(facts) == 0 {
		return AskResult{
			Question: question,
			Answer:   "Todavía no tengo hechos registrados en esta base de conocimiento.",
			Found:    false,
		}
	}

	sort.Slice(facts, func(i, j int) bool {
		return facts[i].CreatedAt.Before(facts[j].CreatedAt)
	})

	return AskResult{
		Question: question,
		Answer:   fmt.Sprintf("Tengo %s en esta conversación.", pluralize(len(facts), "hecho registrado", "hechos registrados")),
		Found:    true,
		Related:  facts,
	}
}

/**
* Facts: returns every active fact currently known inside kbId (loading or creating
* the knowledge base as needed).
* @param kbId string
* @return []*Fact, error
**/
func (s *Engine) Facts(kbId string) ([]*Fact, error) {
	kb, err := s.manager.Load(kbId)
	if err != nil {
		return nil, err
	}

	return kb.ActiveFacts(), nil
}

/**
* Unload: evicts kbId from memory immediately (manual selection), persisting it first
* when the Engine was built with a Store.
* @param kbId string
**/
func (s *Engine) Unload(kbId string) {
	s.manager.Unload(kbId)
}

/**
* IsLoaded: reports whether kbId currently has a knowledge base held in memory.
* @param kbId string
* @return bool
**/
func (s *Engine) IsLoaded(kbId string) bool {
	return s.manager.IsLoaded(kbId)
}
