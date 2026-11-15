package et

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

/**
* SearchTerm: One term of a search text (the operator SEARCH): a word or a "phrase", maybe excluded with -.
**/
type SearchTerm struct {
	Text    string `json:"text"`
	Phrase  bool   `json:"phrase"`
	Exclude bool   `json:"exclude"`
}

/**
* ToSearch: Reads a search text with the web syntax of Postgres' websearch_to_tsquery: words, "phrase", -excluded and
* or between terms. The result is a list of groups joined by OR, each one a list of terms joined by AND; an empty text
* (or one with only or) gives no group, which matches nothing.
* @param text string
* @return [][]SearchTerm
**/
func ToSearch(text string) [][]SearchTerm {
	result := [][]SearchTerm{}
	group := []SearchTerm{}
	closeGroup := func() {
		if len(group) > 0 {
			result = append(result, group)
		}
		group = []SearchTerm{}
	}

	runes := []rune(text)
	for i := 0; i < len(runes); {
		if unicode.IsSpace(runes[i]) {
			i++
			continue
		}
		exclude := false
		if runes[i] == '-' && i+1 < len(runes) && !unicode.IsSpace(runes[i+1]) {
			exclude = true
			i++
		}
		if runes[i] == '"' {
			end := i + 1
			for end < len(runes) && runes[end] != '"' {
				end++
			}
			phrase := strings.Join(strings.Fields(string(runes[i+1:end])), " ")
			if phrase != "" {
				group = append(group, SearchTerm{Text: phrase, Phrase: true, Exclude: exclude})
			}
			i = end + 1
			continue
		}
		end := i
		for end < len(runes) && !unicode.IsSpace(runes[end]) && runes[end] != '"' {
			end++
		}
		word := string(runes[i:end])
		i = end
		if !exclude && strings.EqualFold(word, "or") {
			closeGroup()
			continue
		}
		group = append(group, SearchTerm{Text: word, Exclude: exclude})
	}
	closeGroup()
	return result
}

/**
* FoldText: The text in lowercase and without accents ("Canción" → "cancion"), as the search compares in memory.
* @param text string
* @return string
**/
func FoldText(text string) string {
	result := strings.Builder{}
	for _, r := range norm.NFD.String(text) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

/**
* searchWords: The words of a text, folded: the runs of letters and digits.
* @param text string
* @return []string
**/
func searchWords(text string) []string {
	return strings.FieldsFunc(FoldText(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

/**
* hasWords: Whether words has the run of terms, one after the other.
* @param words, terms []string
* @return bool
**/
func hasWords(words, terms []string) bool {
	if len(terms) == 0 {
		return false
	}
	for i := 0; i+len(terms) <= len(words); i++ {
		match := true
		for j, term := range terms {
			if words[i+j] != term {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

/**
* searchText: The text a value is searched in: a text, the texts of a list joined by a space, or the value as text.
* @param val any
* @return string
**/
func searchText(val any) string {
	switch v := val.(type) {
	case nil:
		return ""
	case string:
		return v
	case []string:
		return strings.Join(v, " ")
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, searchText(item))
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprint(v)
	}
}

/**
* MatchSearch: Whether the value has the search text (ToSearch): every term of one group, as whole words, without case
* nor accents; a phrase is its words one after the other and an excluded term must be missing.
* @param val any, text string
* @return bool
**/
func MatchSearch(val any, text string) bool {
	words := searchWords(searchText(val))
	for _, group := range ToSearch(text) {
		match, used := true, 0
		for _, term := range group {
			terms := searchWords(term.Text)
			if len(terms) == 0 {
				continue
			}
			used++
			if hasWords(words, terms) == term.Exclude {
				match = false
				break
			}
		}
		if match && used > 0 {
			return true
		}
	}
	return false
}

/**
* SearchPattern: The LIKE pattern of a search term: %text% with \ escaping \, %, _ and [ (the engines that do not take
* \ by default need ESCAPE '\').
* @param term SearchTerm
* @return string
**/
func SearchPattern(term SearchTerm) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`, `[`, `\[`)
	return "%" + replacer.Replace(term.Text) + "%"
}

/**
* SearchValue: The text and the lang of the value of a search condition: a text alone, or {"text": "…", "lang": "en"}
* (lang empty when it is not given); false for any other value.
* @param val any
* @return string, string, bool (text, lang, ok)
**/
func SearchValue(val any) (string, string, bool) {
	var obj map[string]any
	switch v := val.(type) {
	case string:
		return v, "", true
	case Json:
		obj = v
	case map[string]any:
		obj = v
	default:
		return "", "", false
	}
	text, ok := obj["text"].(string)
	if !ok {
		return "", "", false
	}
	lang, _ := obj["lang"].(string)
	return text, strings.TrimSpace(lang), true
}
