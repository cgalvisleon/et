package ettp

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
)

const (
	EVENT_RESOLVER_STATUS = "event:resolver:status"
)

type Status int

const (
	TpStatusPending Status = iota
	TpStatusFailed
	TpStatusSuccess
)

func (s Status) String() string {
	switch s {
	case TpStatusPending:
		return "pending"
	case TpStatusFailed:
		return "failed"
	case TpStatusSuccess:
		return "success"
	}

	return "unknown"
}

type Resolver struct {
	*http.Request
	CreatedAt   time.Time                         `json:"created_at"`
	UpdatedAt   time.Time                         `json:"updated_at"`
	Status      Status                            `json:"status"`
	ID          string                            `json:"id"`
	URL         string                            `json:"url"`
	Path        string                            `json:"path"`
	Kind        TypeRouter                        `json:"kind"`
	middlewares []func(http.Handler) http.Handler `json:"-"`
	handlerFn   http.HandlerFunc                  `json:"-"`
}

/**
* newResolver
* @param r *http.Request, solver *Solver, params map[string]string
* @return *Resolver, error
**/
func newResolver(r *http.Request, solver *Solver, params map[string]string) (*Resolver, error) {
	if solver == nil {
		return nil, errors.New(msg.MSG_SOLVER_REQUIRED)
	}

	id := reg.ULID()
	now := utility.Now()
	url := solver.Solver
	for k, v := range params {
		name := strings.Trim(k, "{}")
		r.SetPathValue(name, v)
		url = strings.Replace(url, k, v, 1)
	}

	switch solver.TypeHeader {
	case TpJoinHeader:
		for k, v := range solver.Header {
			idx := slices.IndexFunc(solver.ExcludeHeader, func(h string) bool {
				return h == k
			})

			if idx != -1 {
				continue
			}

			_, ok := r.Header[k]
			if !ok {
				r.Header.Set(k, v)
			}
		}

	case TpReplaceHeader:
		for k, v := range solver.Header {
			idx := slices.IndexFunc(solver.ExcludeHeader, func(h string) bool {
				return h == k
			})

			if idx != -1 {
				continue
			}

			r.Header.Set(k, v)
		}
	}

	result := &Resolver{
		Request:     r,
		CreatedAt:   now,
		UpdatedAt:   now,
		Status:      TpStatusPending,
		ID:          id,
		URL:         url,
		Path:        solver.Path,
		Kind:        solver.Kind,
		middlewares: solver.middlewares,
		handlerFn:   solver.handlerFn,
	}
	result.setStatus(TpStatusPending)

	return result, nil
}

/**
* ToJson
* @return et.Json
**/
func (r *Resolver) ToJson() et.Json {
	return et.Json{
		"created_at": r.CreatedAt,
		"updated_at": r.UpdatedAt,
		"status":     r.Status.String(),
		"id":         r.ID,
		"url":        r.URL,
		"kind":       r.Kind.String(),
	}
}

/**
* setStatus
* @param status Status
**/
func (r *Resolver) setStatus(status Status) {
	r.Status = status
	r.UpdatedAt = utility.Now()
	event.Publish(EVENT_RESOLVER_STATUS, r.ToJson())
}
