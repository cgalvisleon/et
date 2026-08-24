package ia

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/cgalvisleon/et/et"
)

/**
* scoredChunk: A chunk row paired with its cosine-similarity score against a
* query embedding.
**/
type scoredChunk struct {
	chunk et.Json
	score float64
}

/**
* cosineSimilarity: Returns the cosine similarity between a and b, or 0 when
* either vector is empty, of mismatched length, or has zero magnitude.
* @param a []float64, b []float64
* @return float64
**/
func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

/**
* chunkEmbedding: Reads the "embedding" field of a chunk row as []float64.
* Most drivers hand the JSON column back already decoded (a []interface{} of
* numbers); some (e.g. the sqlite driver, whose query builder composes whole
* rows as a single nested JSON object) leave a JSON-typed sub-field as its raw
* encoded string instead of decoding it a second time — this falls back to
* decoding that string directly so retrieval stays driver-agnostic.
* @param c et.Json
* @return []float64
**/
func chunkEmbedding(c et.Json) []float64 {
	if vals := c.ArrayNumber("embedding"); len(vals) > 0 {
		return vals
	}

	if raw, ok := c["embedding"].(string); ok {
		var vals []float64
		if err := json.Unmarshal([]byte(raw), &vals); err == nil {
			return vals
		}
	}

	return nil
}

/**
* topKChunks: Ranks candidates by cosine similarity against query and returns
* the top k, highest score first.
* @param query []float64, candidates []et.Json, k int
* @return []scoredChunk
**/
func topKChunks(query []float64, candidates []et.Json, k int) []scoredChunk {
	scored := make([]scoredChunk, 0, len(candidates))
	for _, c := range candidates {
		embedding := chunkEmbedding(c)
		scored = append(scored, scoredChunk{chunk: c, score: cosineSimilarity(query, embedding)})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if k > 0 && k < len(scored) {
		scored = scored[:k]
	}

	return scored
}
