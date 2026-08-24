package ia

import "strings"

/**
* chunkText: Splits text into overlapping word-based chunks of roughly size
* words each, advancing by (size - overlap) words per chunk. Empty text
* yields no chunks; a text shorter than size yields a single chunk.
* @param text string, size int, overlap int
* @return []string
**/
func chunkText(text string, size, overlap int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	if size <= 0 {
		size = 500
	}
	if overlap < 0 || overlap >= size {
		overlap = 0
	}

	step := size - overlap
	result := make([]string, 0, len(words)/step+1)
	for start := 0; start < len(words); start += step {
		end := min(start+size, len(words))
		result = append(result, strings.Join(words[start:end], " "))
		if end == len(words) {
			break
		}
	}

	return result
}
