package ia

import (
	"context"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

var systemPromptTemplate = `Eres un asistente que SOLO puede responder con base en el CONTEXTO dado.

CONTEXTO:
%s

Reglas obligatorias:
1. Usa unicamente informacion del CONTEXTO.
2. No completes con conocimiento externo.
3. No hagas suposiciones.
4. Si la respuesta no esta explicitamente en el CONTEXTO, responde exactamente:
"` + MSG_NO_ANSWER + `"
`

/**
* client: Returns an OpenAI client authenticated with the Rag's configured API key.
* @return openai.Client
**/
func (s *Rag) client() openai.Client {
	return openai.NewClient(option.WithAPIKey(s.cnf.ApiKey))
}

/**
* embed: Generates the embedding vector for text using the configured embedding model.
* @param ctx context.Context, text string
* @return []float64, error
**/
func (s *Rag) embed(ctx context.Context, text string) ([]float64, error) {
	client := s.client()
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: s.cnf.EmbeddingModel,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
	})
	if err != nil {
		return nil, err
	}

	return resp.Data[0].Embedding, nil
}

/**
* answer: Asks the configured chat model to answer question using only the
* given context chunks, following the RAG system prompt.
* @param ctx context.Context, question string, contextChunks []string
* @return string, error
**/
func (s *Rag) answer(ctx context.Context, question string, contextChunks []string) (string, error) {
	systemPrompt := systemPromptTemplate
	if len(contextChunks) > 0 {
		systemPrompt = strings.Replace(systemPromptTemplate, "%s", strings.Join(contextChunks, "\n---\n"), 1)
	} else {
		systemPrompt = strings.Replace(systemPromptTemplate, "%s", "(sin contexto disponible)", 1)
	}

	client := s.client()
	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: s.cnf.ChatModel,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(question),
		},
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return MSG_NO_ANSWER, nil
	}

	return resp.Choices[0].Message.Content, nil
}
