package ia

import "context"

// LLM is the contract Engine.Ask uses to ask a local language model to reason over
// retrieved facts and produce a natural-language answer. It is an optional
// enhancement layer, not a hard dependency — Engine works exactly as before when no
// LLM is configured (see Engine.UseLLM). OllamaLLM (ollama.go) is the reference
// implementation; callers may supply their own (e.g. a test double) as long as it
// satisfies this interface.
// @param ctx context.Context, prompt string
// @return string, error
type LLM interface {
	Complete(ctx context.Context, prompt string) (string, error)
}
