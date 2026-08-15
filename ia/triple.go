package ia

import "strings"

// Triple: a lightweight subject-predicate-object extraction from one clause of a
// Fact's statement, giving attribute-level anchors ("who did what to what") on top
// of the raw sentence text — used to spot real contradictions (same
// subject+predicate, different object) and to ground the LLM/retrieval with
// structured context instead of prose alone.
// @param Subject, Predicate, Object string
type Triple struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
}

// tripleVerbs: raw (accent-stripped, lowercase) verb forms extractTriples looks for
// as a clause's predicate anchor. A flat set of common conjugated forms rather than a
// lemmatizer — this is a lightweight heuristic, not a real dependency parser, and is
// meant to be extended as new phrasings show up in real usage.
var tripleVerbs = map[string]bool{
	"es": true, "fue": true, "son": true, "era": true,
	"esta": true, "estan": true, "estuve": true, "estuvo": true,
	"tengo": true, "tiene": true, "tenia": true,
	"compre": true, "compro": true, "vendi": true, "vendio": true,
	"pague": true, "pago": true, "firme": true, "firmo": true,
	"entregue": true, "entrego": true, "recibi": true, "recibio": true,
	"hable": true, "hablo": true, "vi": true, "vio": true,
	"llegue": true, "llego": true, "fui": true,
	"deposite": true, "deposito": true, "cene": true, "ceno": true,
	"termine": true, "termino": true,
}

// tripleStopwords: leading prepositions/articles trimmed off subject/object phrases
// so they start at meaningful content instead of a function word.
var tripleStopwords = map[string]bool{
	"en": true, "el": true, "la": true, "los": true, "las": true, "a": true,
	"de": true, "al": true, "del": true, "con": true, "por": true, "para": true,
}

/**
* extractTriples: rule-based subject-predicate-object extraction over statement's
* clauses (split on the token "y"). Within each clause, the predicate is the first
* token matching tripleVerbs; the subject is whatever precedes it (defaulting to
* "yo" when empty or blank, since these facts are almost always first-person); the
* object is whatever follows, both trimmed of leading prepositions/articles via
* tripleStopwords. Clauses without a recognizable verb are skipped.
* @param statement string
* @return []Triple
**/
func extractTriples(statement string) []Triple {
	tokens := tokenize(normalizeStatement(statement))
	if len(tokens) == 0 {
		return nil
	}

	var triples []Triple
	start := 0
	for i := 0; i <= len(tokens); i++ {
		if i == len(tokens) || tokens[i] == "y" {
			if triple, ok := extractClauseTriple(tokens[start:i]); ok {
				triples = append(triples, triple)
			}
			start = i + 1
		}
	}

	return triples
}

/**
* extractClauseTriple: applies extractTriples' heuristic to a single clause's tokens.
* @param tokens []string
* @return Triple, bool
**/
func extractClauseTriple(tokens []string) (Triple, bool) {
	verbIdx := -1
	for i, tok := range tokens {
		if tripleVerbs[tok] {
			verbIdx = i
			break
		}
	}
	if verbIdx == -1 {
		return Triple{}, false
	}

	subject := strings.Join(trimLeadingStopwords(tokens[:verbIdx]), " ")
	if subject == "" {
		subject = "yo"
	}
	object := strings.Join(trimLeadingStopwords(tokens[verbIdx+1:]), " ")

	return Triple{Subject: subject, Predicate: tokens[verbIdx], Object: object}, true
}

/**
* trimLeadingStopwords: drops leading tokens found in tripleStopwords (keeping at
* least the last token), so the phrase starts at its first meaningful word.
* @param tokens []string
* @return []string
**/
func trimLeadingStopwords(tokens []string) []string {
	i := 0
	for i < len(tokens)-1 && tripleStopwords[tokens[i]] {
		i++
	}

	return tokens[i:]
}

// tripleAnchorBoost is added to a candidate fact's similarity score in
// RelevantFacts when it shares a (Subject, Predicate) anchor with the query, so
// attribute-level matches ("cuanto pague" -> Predicate "pague") rank higher even
// when the wording otherwise differs.
const tripleAnchorBoost = 0.15

/**
* tripleShareAnchor: reports whether a and b share at least one (Subject, Predicate)
* pair — i.e. they're about the same "who did what", regardless of Object.
* @param a, b []Triple
* @return bool
**/
func tripleShareAnchor(a, b []Triple) bool {
	for _, ta := range a {
		for _, tb := range b {
			if ta.Subject == tb.Subject && ta.Predicate == tb.Predicate {
				return true
			}
		}
	}

	return false
}

/**
* triplesContradict: reports whether a and b share a (Subject, Predicate) anchor but
* disagree on Object — a structured signal that two statements make conflicting
* claims about the same attribute, sharper than a heuristic based on the presence of
* a negation word.
* @param a, b []Triple
* @return bool
**/
func triplesContradict(a, b []Triple) bool {
	for _, ta := range a {
		for _, tb := range b {
			if ta.Subject == tb.Subject && ta.Predicate == tb.Predicate && ta.Object != tb.Object {
				return true
			}
		}
	}

	return false
}
