package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultOllamaTimeout bounds an OllamaLLM.Complete call when NewOllamaLLM is given
// timeout <= 0.
const DefaultOllamaTimeout = 10 * time.Second

// OllamaLLM is an LLM (see llm.go) backed by a local Ollama server's REST API
// (POST {host}/api/generate). It talks to Ollama over plain net/http — no extra
// dependency in go.mod — so it works with Ollama running natively or, as this repo
// recommends (see docker-compose.yml), in a container with its port published to
// the host.
// @param host, model string
type OllamaLLM struct {
	host   string
	model  string
	client *http.Client
}

/**
* NewOllamaLLM: builds an OllamaLLM targeting host (e.g. "http://localhost:11434")
* and model (e.g. "llama3.2"), bounding every Complete call to timeout
* (DefaultOllamaTimeout when <= 0).
* @param host, model string, timeout time.Duration
* @return *OllamaLLM
**/
func NewOllamaLLM(host, model string, timeout time.Duration) *OllamaLLM {
	if timeout <= 0 {
		timeout = DefaultOllamaTimeout
	}

	return &OllamaLLM{
		host:   strings.TrimRight(host, "/"),
		model:  model,
		client: &http.Client{Timeout: timeout},
	}
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

/**
* Complete: sends prompt to Ollama's /api/generate endpoint (non-streaming) and
* returns the generated text.
* @param ctx context.Context, prompt string
* @return string, error
**/
func (s *OllamaLLM) Complete(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(ollamaGenerateRequest{Model: s.model, Prompt: prompt, Stream: false})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.host+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("ollama: unexpected status %d: %s", res.StatusCode, string(data))
	}

	var result ollamaGenerateResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Response, nil
}
