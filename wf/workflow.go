package workflow

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
)

const (
	packageName = "workflow"
)

type Store interface {
	Set(collection, id, tenantId, ownerId string, obj any, userId string) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
	SetModule(module string, source any) error
	GetModule(module string, source any) (bool, error)
	DeleteModule(module string) error
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
