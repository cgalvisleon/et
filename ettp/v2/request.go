package ettp

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
)

type Request struct {
	*http.Request
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Id          string    `json:"id"`
	Kind        TypeApi   `json:"kind"`
	Path        string    `json:"path"`
	URL         string    `json:"url"`
	Version     int       `json:"version"`
	PackageName string    `json:"package_name"`
}

func NewRequest(r *http.Request, solver *Solver, params map[string]string) *Request {
	id := reg.GetUlId("request", "")
	now := utility.NowTime()
	url := solver.Solver
	for k, v := range params {
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

	result := &Request{
		Request:     r,
		CreatedAt:   now,
		UpdatedAt:   now,
		Id:          id,
		Kind:        solver.Kind,
		Path:        solver.Path,
		URL:         url,
		Version:     solver.Version,
		PackageName: solver.PackageName,
	}

	return result
}
