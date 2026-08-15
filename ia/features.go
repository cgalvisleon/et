package ia

import "regexp"

var (
	digitRe    = regexp.MustCompile(`\d+`)
	negationRe = regexp.MustCompile(`\bno\b`)
)

// Lexicon holds the word lists ExtractFeatures uses to score the linguistic markers
// classically associated with deception (Newman et al. 2003; Ott et al. 2011):
// fewer first-person/self-references, more other-references and negative-emotion
// words, fewer exclusive/differentiation words, and less certainty language. Words
// must be stored without accents/punctuation, matching normalizeStatement's output.
type Lexicon struct {
	FirstPerson map[string]bool
	ThirdPerson map[string]bool
	Negative    map[string]bool
	Positive    map[string]bool
	Exclusive   map[string]bool
	Certainty   map[string]bool
	Tentative   map[string]bool
}

/**
* newWordSet: builds a lookup set from a list of words.
* @param words ...string
* @return map[string]bool
**/
func newWordSet(words ...string) map[string]bool {
	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[w] = true
	}

	return set
}

/**
* DefaultLexiconES: a small hand-curated Spanish lexicon covering the marker
* categories ExtractFeatures reads. It is intentionally compact — callers training on
* a specific domain or dataset should build and pass their own Lexicon instead.
* @return *Lexicon
**/
func DefaultLexiconES() *Lexicon {
	return &Lexicon{
		FirstPerson: newWordSet("yo", "me", "mi", "mis", "nosotros", "nuestro", "nuestra"),
		ThirdPerson: newWordSet("el", "ella", "ellos", "ellas", "su", "sus", "le", "les"),
		Negative:    newWordSet("mal", "malo", "mala", "triste", "odio", "miedo", "problema", "nunca"),
		Positive:    newWordSet("bien", "bueno", "buena", "feliz", "amor", "genial", "excelente"),
		Exclusive:   newWordSet("pero", "excepto", "sin", "embargo", "aunque", "salvo", "menos"),
		Certainty:   newWordSet("siempre", "definitivamente", "seguro", "claro", "obvio", "totalmente"),
		Tentative:   newWordSet("quizas", "quiza", "creo", "posiblemente", "parece", "probablemente", "tal", "vez"),
	}
}

// FeatureCount is the fixed length of the vector returned by Features.Vector.
const FeatureCount = 12

// Features: the fixed-length numeric feature vector fed into the classifier's model.
// @param WordCount, AvgWordLength, FirstPersonRatio, ThirdPersonRatio, NegativeRatio,
// PositiveRatio, ExclusiveRatio, CertaintyRatio, TentativeRatio, DetailCount,
// MaxKBSimilarity, ContradictsKB float64
type Features struct {
	WordCount        float64
	AvgWordLength    float64
	FirstPersonRatio float64
	ThirdPersonRatio float64
	NegativeRatio    float64
	PositiveRatio    float64
	ExclusiveRatio   float64
	CertaintyRatio   float64
	TentativeRatio   float64
	DetailCount      float64
	MaxKBSimilarity  float64
	ContradictsKB    float64
}

/**
* Vector: returns the Features in a fixed field order, as expected by Model.Predict
* and Model.Train.
* @return []float64
**/
func (s Features) Vector() []float64 {
	return []float64{
		s.WordCount,
		s.AvgWordLength,
		s.FirstPersonRatio,
		s.ThirdPersonRatio,
		s.NegativeRatio,
		s.PositiveRatio,
		s.ExclusiveRatio,
		s.CertaintyRatio,
		s.TentativeRatio,
		s.DetailCount,
		s.MaxKBSimilarity,
		s.ContradictsKB,
	}
}

/**
* ExtractFeatures: computes the linguistic feature vector for statement. When kb is
* not nil, it also scores statement against the closest active fact already known in
* kb (via the token index, so it does not scan the whole knowledge base) to capture
* how novel or contradictory statement is relative to what is already believed true.
* A nil lex falls back to DefaultLexiconES.
* @param statement string, kb *KnowledgeBase, lex *Lexicon
* @return Features
**/
func ExtractFeatures(statement string, kb *KnowledgeBase, lex *Lexicon) Features {
	if lex == nil {
		lex = DefaultLexiconES()
	}

	normalized := normalizeStatement(statement)
	tokens := tokenize(normalized)
	n := float64(len(tokens))

	features := Features{WordCount: n}
	if n == 0 {
		return features
	}

	totalLen := 0
	var firstPerson, thirdPerson, negative, positive, exclusive, certainty, tentative float64
	for _, tok := range tokens {
		totalLen += len(tok)

		if lex.FirstPerson[tok] {
			firstPerson++
		}
		if lex.ThirdPerson[tok] {
			thirdPerson++
		}
		if lex.Negative[tok] {
			negative++
		}
		if lex.Positive[tok] {
			positive++
		}
		if lex.Exclusive[tok] {
			exclusive++
		}
		if lex.Certainty[tok] {
			certainty++
		}
		if lex.Tentative[tok] {
			tentative++
		}
	}

	features.AvgWordLength = float64(totalLen) / n
	features.FirstPersonRatio = firstPerson / n
	features.ThirdPersonRatio = thirdPerson / n
	features.NegativeRatio = negative / n
	features.PositiveRatio = positive / n
	features.ExclusiveRatio = exclusive / n
	features.CertaintyRatio = certainty / n
	features.TentativeRatio = tentative / n
	features.DetailCount = float64(len(digitRe.FindAllString(normalized, -1)))

	if kb != nil {
		scoreAgainstKB(statement, normalized, kb, &features)
	}

	return features
}

/**
* scoreAgainstKB: fills MaxKBSimilarity/ContradictsKB by comparing statement against
* the closest active fact kb already knows, using kb.FindSimilar as a token-index
* candidate lookup instead of scanning every fact.
* @param statement, normalized string, kb *KnowledgeBase, features *Features
**/
func scoreAgainstKB(statement, normalized string, kb *KnowledgeBase, features *Features) {
	bestFact, best := ClosestFact(kb, statement)

	features.MaxKBSimilarity = best
	if bestFact == nil || best < 0.3 {
		return
	}

	// Two contradiction signals: a structured one (same subject+predicate anchor but
	// a different object — e.g. "yo pague" with two different amounts) and, as a
	// fallback for clauses extractTriples couldn't anchor, the cruder heuristic that
	// statements about the same topic where exactly one side carries a negation
	// word are flagged as conflicting.
	if triplesContradict(extractTriples(statement), bestFact.Triples) {
		features.ContradictsKB = 1
		return
	}

	hasNegation := negationRe.MatchString(normalized)
	factHasNegation := negationRe.MatchString(bestFact.Normalized)
	if hasNegation != factHasNegation {
		features.ContradictsKB = 1
	}
}
