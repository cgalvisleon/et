package ia

import (
	"errors"
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
