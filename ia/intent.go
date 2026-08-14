package ia

// listFactsThreshold is how similar a question needs to be to one of
// listFactsPhrases (via StatementSimilarity) to be treated as a request to enumerate
// the knowledge base, rather than a topical lookup. Higher than MinAskSimilarity
// since these are short canonical phrases where a genuine match scores highly.
const listFactsThreshold = 0.55

// listFactsPhrases: canonical Spanish phrasings of "tell me everything you know",
// used by isListFactsIntent to recognize meta-questions about the knowledge base
// itself instead of a specific fact.
var listFactsPhrases = []string{
	"que hechos tienes",
	"que sabes",
	"que informacion tienes",
	"que tienes en la base de conocimiento",
	"que conoces",
	"muestrame lo que sabes",
	"resumen de lo que sabes",
	"lista los hechos",
	"cuantos hechos tienes",
	"que hay en la base de conocimiento",
}

// greetingThreshold mirrors listFactsThreshold's reasoning: greetings are short
// canonical phrases, so a genuine match scores highly.
const greetingThreshold = 0.55

// greetingPhrases: canonical Spanish greetings/small talk recognized by
// isGreetingIntent, so Ask can reply warmly instead of "no tengo información".
var greetingPhrases = []string{
	"hola",
	"buenas",
	"buenos dias",
	"buenas tardes",
	"buenas noches",
	"como estas",
	"gracias",
}

/**
* matchesPhraseSet: reports whether text is close enough (StatementSimilarity, at
* least threshold) to any phrase in phrases. Shared by isListFactsIntent and
* isGreetingIntent to recognize a fixed set of canonical question/phrase shapes.
* @param text string, phrases []string, threshold float64
* @return bool
**/
func matchesPhraseSet(text string, phrases []string, threshold float64) bool {
	for _, phrase := range phrases {
		if StatementSimilarity(text, phrase) >= threshold {
			return true
		}
	}

	return false
}

/**
* isListFactsIntent: reports whether question is close enough to one of
* listFactsPhrases to be treated as a request to enumerate every known fact, rather
* than a search for one in particular.
* @param question string
* @return bool
**/
func isListFactsIntent(question string) bool {
	return matchesPhraseSet(question, listFactsPhrases, listFactsThreshold)
}

/**
* isGreetingIntent: reports whether question is close enough to one of
* greetingPhrases to be treated as small talk rather than a real question about the
* knowledge base.
* @param question string
* @return bool
**/
func isGreetingIntent(question string) bool {
	return matchesPhraseSet(question, greetingPhrases, greetingThreshold)
}
