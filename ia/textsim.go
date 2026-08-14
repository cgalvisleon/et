package ia

/**
* Levenshtein: computes the edit distance (insertions, deletions, substitutions)
* between a and b, operating on runes so accented/multi-byte characters count once.
* @param a, b string
* @return int
**/
func Levenshtein(a, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	la, lb := len(ra), len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

/**
* LevenshteinRatio: converts Levenshtein into a similarity score in [0,1], where 1
* means the strings are identical and 0 means they share nothing.
* @param a, b string
* @return float64
**/
func LevenshteinRatio(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 && len(rb) == 0 {
		return 1
	}

	maxLen := max(len(ra), len(rb))
	if maxLen == 0 {
		return 1
	}

	dist := Levenshtein(a, b)
	return 1 - float64(dist)/float64(maxLen)
}

/**
* JaccardSimilarity: computes the Jaccard index (intersection over union) between the
* token sets of a and b, in [0,1].
* @param a, b []string
* @return float64
**/
func JaccardSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}

	setA := toSet(a)
	setB := toSet(b)

	intersection := 0
	for token := range setA {
		if setB[token] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

/**
* toSet: builds a lookup set from a slice of tokens.
* @param tokens []string
* @return map[string]bool
**/
func toSet(tokens []string) map[string]bool {
	set := make(map[string]bool, len(tokens))
	for _, token := range tokens {
		set[token] = true
	}

	return set
}

/**
* StatementSimilarity: blends token overlap (Jaccard) with character-level similarity
* (Levenshtein ratio) to score how close two raw statements are once normalized, in
* [0,1]. Jaccard is weighted higher since it is more robust to word reordering.
* @param a, b string
* @return float64
**/
func StatementSimilarity(a, b string) float64 {
	na := normalizeStatement(a)
	nb := normalizeStatement(b)

	jaccard := JaccardSimilarity(tokenize(na), tokenize(nb))
	lev := LevenshteinRatio(na, nb)

	return 0.6*jaccard + 0.4*lev
}
