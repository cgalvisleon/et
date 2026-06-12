package workflow

import (
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
)

const (
	packageName = "workflow"
)

type Store interface {
	Set(collection, id, tenantId, projectId string, obj any, userId string) error
	// By Collection
	GetByCollection(collection, id string, dest any) (bool, error)
	DeleteByCollection(collection, id string) error
	// By Id
	Get(id string, dest any) (bool, error)
	Delete(id string) error
	// By Query
	Query(query et.Json) (et.Items, error)
	// By Module
	SetModule(module string, source any) error
	GetModule(module string, source any) (bool, error)
	DeleteModule(module string) error
	// Series by tag
	GetCode(tag string) (string, error)
}

type WorkFlow struct {
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
	TenantId    string               `json:"tenant_id"`
	ID          string               `json:"id"`
	Flows       map[string]*Flow     `json:"flows"`
	Instances   map[string]*Instance `json:"-"`
	bindings    map[string]any       `json:"-"`
	muFlows     sync.Mutex           `json:"-"`
	muInstances sync.Mutex           `json:"-"`
	store       Store                `json:"-"`
	metrics     cache.Metrics        `json:"-"`
	isDebug     bool                 `json:"-"`
}

/**
* getFlow
* @param tag string
* @return *Flow, error
**/
func (s *WorkFlow) getFlow(id, userId string) (*Flow, error) {
	flow, exists := s.Flows[id]
	if exists {
		return flow, nil
	}

	flow, err := s.loadFlow(id, userId)
	if err != nil {
		return nil, err
	}

	if flow == nil {
		return nil, errors.New(MSG_FLOW_NOT_FOUND)
	}

	return flow, nil
}
