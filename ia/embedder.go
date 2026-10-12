package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

/**
* Embedder: Turns texts into embedding vectors, one per text and in the same order. Each caller brings its own
* (its provider, key and model), so one Rag can index with many providers.
**/
type Embedder func(ctx context.Context, texts []string) ([][]float64, error)

// embedTimeout: how long an HTTP embedder waits for the provider
const embedTimeout = 60 * time.Second

var embedClient = &http.Client{Timeout: embedTimeout}

/**
* HttpEmbedder: An Embedder for an OpenAI-compatible provider: POST {baseUrl}/embeddings with {model, input} and
* Authorization: Bearer apiKey (no header without a key); the vectors come back in data[].embedding, ordered by index.
* @param baseUrl, apiKey, model string
* @return Embedder
**/
func HttpEmbedder(baseUrl, apiKey, model string) Embedder {
	url := strings.TrimRight(baseUrl, "/") + "/embeddings"
	return func(ctx context.Context, texts []string) ([][]float64, error) {
		if len(texts) == 0 {
			return [][]float64{}, nil
		}

		payload, err := json.Marshal(map[string]any{"model": model, "input": texts})
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		res, err := embedClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf(MSG_EMBEDDINGS_FAILED, err.Error())
		}
		defer res.Body.Close()

		raw, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		if res.StatusCode >= 400 {
			detail := fmt.Sprintf("%d %s", res.StatusCode, strings.TrimSpace(string(raw)))
			return nil, fmt.Errorf(MSG_EMBEDDINGS_FAILED, detail)
		}

		var body struct {
			Data []struct {
				Index     int       `json:"index"`
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf(MSG_EMBEDDINGS_FAILED, err.Error())
		}

		if len(body.Data) != len(texts) {
			return nil, fmt.Errorf(MSG_EMBEDDINGS_COUNT, len(body.Data), len(texts))
		}

		sort.SliceStable(body.Data, func(i, j int) bool {
			return body.Data[i].Index < body.Data[j].Index
		})

		result := make([][]float64, len(body.Data))
		for i, item := range body.Data {
			result[i] = item.Embedding
		}

		return result, nil
	}
}
