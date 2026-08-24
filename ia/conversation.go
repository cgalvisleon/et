package ia

import (
	"context"
	"errors"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

const (
	roleUser      = "user"
	roleAssistant = "assistant"
	titleMaxLen   = 80
)

/**
* truncate: Returns s cut down to at most n runes, so a long question does not
* blow out the conversation title column.
* @param s string, n int
* @return string
**/
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

/**
* Ask: Answers question using Retrieval-Augmented Generation: embeds the
* question, retrieves the most relevant chunks previously ingested under the
* given tenant and project, asks the chat model to answer using only that
* context, and persists both the question and the answer as messages of
* conversationId (a new conversation is created when conversationId is empty).
* @param ctx context.Context, tenantId, projectId, conversationId, userId, question string
* @return et.Json, error
**/
func (s *Rag) Ask(ctx context.Context, tenantId, projectId, conversationId, userId, question string) (et.Json, error) {
	if !utility.ValidStr(tenantId, 0, []string{""}) {
		return nil, errors.New(MSG_TENANT_ID_REQUIRED)
	}
	if !utility.ValidStr(projectId, 0, []string{""}) {
		return nil, errors.New(MSG_PROJECT_ID_REQUIRED)
	}
	if !utility.ValidStr(question, 1, []string{}) {
		return nil, errors.New(MSG_QUESTION_REQUIRED)
	}

	now := timezone.Now()
	if conversationId == "" {
		conversationId = reg.UUID()
		_, err := s.conversations.Insert(et.Json{
			"id":         conversationId,
			"tenant_id":  tenantId,
			"project_id": projectId,
			"user_id":    userId,
			"title":      truncate(question, titleMaxLen),
			"created_at": now,
			"updated_at": now,
		}).ExecTx(nil)
		if err != nil {
			return nil, err
		}
	}

	queryEmbedding, err := s.embedFn(ctx, question)
	if err != nil {
		return nil, err
	}

	candidates, err := s.chunks.
		Where(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId)).
		All()
	if err != nil {
		return nil, err
	}

	top := topKChunks(queryEmbedding, candidates.Result, s.cnf.TopK)
	contextChunks := make([]string, 0, len(top))
	sources := make([]any, 0, len(top))
	for _, c := range top {
		if c.score <= 0 {
			continue
		}
		contextChunks = append(contextChunks, c.chunk.Str("content"))
		sources = append(sources, et.Json{
			"document_id": c.chunk.Str("document_id"),
			"chunk_id":    c.chunk.Str("id"),
			"score":       c.score,
		})
	}

	answer, err := s.answerFn(ctx, question, contextChunks)
	if err != nil {
		return nil, err
	}

	if _, err := s.messages.Insert(et.Json{
		"id":              reg.UUID(),
		"tenant_id":       tenantId,
		"project_id":      projectId,
		"conversation_id": conversationId,
		"role":            roleUser,
		"content":         question,
		"sources":         []any{},
		"created_at":      now,
		"updated_at":      now,
	}).ExecTx(nil); err != nil {
		return nil, err
	}

	answerNow := timezone.Now()
	if _, err := s.messages.Insert(et.Json{
		"id":              reg.UUID(),
		"tenant_id":       tenantId,
		"project_id":      projectId,
		"conversation_id": conversationId,
		"role":            roleAssistant,
		"content":         answer,
		"sources":         sources,
		"created_at":      answerNow,
		"updated_at":      answerNow,
	}).ExecTx(nil); err != nil {
		return nil, err
	}

	return et.Json{
		"conversation_id": conversationId,
		"answer":          answer,
		"sources":         sources,
	}, nil
}

/**
* ListMessages: Returns the message history of a conversation, scoped to the
* given tenant and project, oldest first.
* @param tenantId, projectId, conversationId string
* @return et.Items, error
**/
func (s *Rag) ListMessages(tenantId, projectId, conversationId string) (et.Items, error) {
	return s.messages.
		Where(jsql.Eq(jsql.TENANT_ID, tenantId)).
		And(jsql.Eq(jsql.PROJECT_ID, projectId)).
		And(jsql.Eq("conversation_id", conversationId)).
		OrderBy("created_at", true).
		All()
}
