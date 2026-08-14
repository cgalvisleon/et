package ia

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
)

// FactStatus represents the lifecycle state of a Fact inside a KnowledgeBase.
type FactStatus string

const (
	FactActive     FactStatus = "active"
	FactSuperseded FactStatus = "superseded"
	FactRetracted  FactStatus = "retracted"
)

var punctuationRe = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

// Fact: a single piece of knowledge learned inside a KnowledgeBase. A Fact can be
// superseded by a newer version of itself when more context changes what is known
// to be true, instead of being mutated in place.
// @param ID, Statement, Normalized string, Confidence float64, Status FactStatus,
// Version int, SupersedesID string, Context et.Json, CreatedAt, UpdatedAt time.Time
type Fact struct {
	ID           string     `json:"id"`
	Statement    string     `json:"statement"`
	Normalized   string     `json:"normalized"`
	Confidence   float64    `json:"confidence"`
	Status       FactStatus `json:"status"`
	Version      int        `json:"version"`
	SupersedesID string     `json:"supersedes_id"`
	Context      et.Json    `json:"context"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

/**
* normalizeStatement: lower-cases, strips accents and punctuation, and collapses
* whitespace so statements can be compared and tokenized consistently.
* @param statement string
* @return string
**/
func normalizeStatement(statement string) string {
	s := strs.RemoveAcents(strs.Lowcase(strs.Trim(statement)))
	s = punctuationRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

/**
* tokenize: splits a normalized statement into its word tokens.
* @param normalized string
* @return []string
**/
func tokenize(normalized string) []string {
	if normalized == "" {
		return nil
	}

	return strings.Fields(normalized)
}

// KnowledgeBase: an in-memory, concurrency-safe collection of Facts scoped to a
// single conversation/context id. Facts are indexed by id for O(1) access and by
// token for fast candidate lookup when searching for similar or contradicting
// statements (FindSimilar), avoiding a full scan on every check.
// @param ID string, Facts map[string]*Fact
type KnowledgeBase struct {
	ID         string           `json:"id"`
	Facts      map[string]*Fact `json:"facts"`
	CreatedAt  time.Time        `json:"created_at"`
	mu         sync.RWMutex     `json:"-"`
	tokenIndex map[string]map[string]bool
}

/**
* NewKnowledgeBase: builds an empty KnowledgeBase for the given id.
* @param id string
* @return *KnowledgeBase
**/
func NewKnowledgeBase(id string) *KnowledgeBase {
	return &KnowledgeBase{
		ID:         id,
		Facts:      make(map[string]*Fact),
		CreatedAt:  timezone.Now(),
		tokenIndex: make(map[string]map[string]bool),
	}
}

/**
* indexFact: adds a Fact's tokens to the inverted index. Caller must hold the lock.
* @param fact *Fact
**/
func (s *KnowledgeBase) indexFact(fact *Fact) {
	for _, token := range tokenize(fact.Normalized) {
		ids, ok := s.tokenIndex[token]
		if !ok {
			ids = make(map[string]bool)
			s.tokenIndex[token] = ids
		}
		ids[fact.ID] = true
	}
}

/**
* unindexFact: removes a Fact's tokens from the inverted index. Caller must hold the lock.
* @param fact *Fact
**/
func (s *KnowledgeBase) unindexFact(fact *Fact) {
	for _, token := range tokenize(fact.Normalized) {
		ids, ok := s.tokenIndex[token]
		if !ok {
			continue
		}
		delete(ids, fact.ID)
		if len(ids) == 0 {
			delete(s.tokenIndex, token)
		}
	}
}

/**
* AddFact: learns a brand new fact with FactActive status and version 1.
* @param statement string, confidence float64, ctx et.Json
* @return *Fact, error
**/
func (s *KnowledgeBase) AddFact(statement string, confidence float64, ctx et.Json) (*Fact, error) {
	if strs.IsEmpty(statement) {
		return nil, errors.New(MSG_STATEMENT_EMPTY)
	}

	now := timezone.Now()
	fact := &Fact{
		ID:         reg.UUID(),
		Statement:  statement,
		Normalized: normalizeStatement(statement),
		Confidence: confidence,
		Status:     FactActive,
		Version:    1,
		Context:    ctx,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.Facts[fact.ID] = fact
	s.indexFact(fact)

	return fact, nil
}

/**
* UpdateFact: records a new version of an existing fact, marking the previous one as
* superseded rather than overwriting it, so the KnowledgeBase keeps a history of how
* a truth changed as more context arrived.
* @param id, statement string, confidence float64, ctx et.Json
* @return *Fact, error
**/
func (s *KnowledgeBase) UpdateFact(id string, statement string, confidence float64, ctx et.Json) (*Fact, error) {
	if strs.IsEmpty(statement) {
		return nil, errors.New(MSG_STATEMENT_EMPTY)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	prev, ok := s.Facts[id]
	if !ok {
		return nil, errors.New(MSG_FACT_NOT_FOUND)
	}

	now := timezone.Now()
	prev.Status = FactSuperseded
	prev.UpdatedAt = now
	s.unindexFact(prev)

	next := &Fact{
		ID:           reg.UUID(),
		Statement:    statement,
		Normalized:   normalizeStatement(statement),
		Confidence:   confidence,
		Status:       FactActive,
		Version:      prev.Version + 1,
		SupersedesID: prev.ID,
		Context:      ctx,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.Facts[next.ID] = next
	s.indexFact(next)

	return next, nil
}

/**
* RetractFact: marks a fact as retracted without deleting its history.
* @param id string
* @return error
**/
func (s *KnowledgeBase) RetractFact(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fact, ok := s.Facts[id]
	if !ok {
		return errors.New(MSG_FACT_NOT_FOUND)
	}

	fact.Status = FactRetracted
	fact.UpdatedAt = timezone.Now()
	s.unindexFact(fact)

	return nil
}

/**
* GetFact: returns the fact stored under id, if any.
* @param id string
* @return *Fact, bool
**/
func (s *KnowledgeBase) GetFact(id string) (*Fact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fact, ok := s.Facts[id]
	return fact, ok
}

/**
* Facts: returns every active fact currently held in the knowledge base.
* @return []*Fact
**/
func (s *KnowledgeBase) ActiveFacts() []*Fact {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Fact, 0, len(s.Facts))
	for _, fact := range s.Facts {
		if fact.Status == FactActive {
			result = append(result, fact)
		}
	}

	return result
}

/**
* FindSimilar: returns the active facts that share at least one token with statement,
* using the inverted index instead of scanning every fact in the knowledge base.
* @param statement string
* @return []*Fact
**/
func (s *KnowledgeBase) FindSimilar(statement string) []*Fact {
	normalized := normalizeStatement(statement)
	tokens := tokenize(normalized)

	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	result := make([]*Fact, 0)
	for _, token := range tokens {
		for factId := range s.tokenIndex[token] {
			if seen[factId] {
				continue
			}
			seen[factId] = true

			fact, ok := s.Facts[factId]
			if ok && fact.Status == FactActive {
				result = append(result, fact)
			}
		}
	}

	return result
}

/**
* Len: returns the total number of facts (any status) held in the knowledge base.
* @return int
**/
func (s *KnowledgeBase) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.Facts)
}

/**
* ClosestFact: returns the active fact in kb most similar to statement (by
* StatementSimilarity) among kb.FindSimilar's candidates, along with its similarity
* score. Returns (nil, 0) when kb is nil or has no matching candidates.
* @param kb *KnowledgeBase, statement string
* @return *Fact, float64
**/
func ClosestFact(kb *KnowledgeBase, statement string) (*Fact, float64) {
	facts := RelevantFacts(kb, statement, 1, 0)
	if len(facts) == 0 {
		return nil, 0
	}

	return facts[0], StatementSimilarity(statement, facts[0].Statement)
}

/**
* RelevantFacts: returns kb's active facts most similar to statement (by
* StatementSimilarity), scoring at least minScore, sorted from most to least similar
* and capped at limit results (no cap when limit <= 0). Uses kb.FindSimilar for
* candidates instead of scanning every fact. Returns nil when kb is nil.
* @param kb *KnowledgeBase, statement string, limit int, minScore float64
* @return []*Fact
**/
func RelevantFacts(kb *KnowledgeBase, statement string, limit int, minScore float64) []*Fact {
	if kb == nil {
		return nil
	}

	type scoredFact struct {
		fact  *Fact
		score float64
	}

	candidates := kb.FindSimilar(statement)
	scored := make([]scoredFact, 0, len(candidates))
	for _, fact := range candidates {
		score := StatementSimilarity(statement, fact.Statement)
		if score >= minScore {
			scored = append(scored, scoredFact{fact, score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}

	result := make([]*Fact, len(scored))
	for i, sf := range scored {
		result[i] = sf.fact
	}

	return result
}
