package jtenant

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

const packageName = "jtenant"

type ServerDB interface {
	NewDB(name string) (*jsql.DB, error)
	LoadDB(name string) (*jsql.DB, error)
}

type ServerStorage interface {
	NewFolder(name string) (et.Item, error)
	DeleteFolder(name string) (et.Item, error)
	UploadFile(paths, fileName string, file []byte) (et.Item, error)
	DownloadFile(paths, fileName string) (et.Item, error)
	DeleteFile(paths, fileName string) (et.Item, error)
}

type Store interface {
	Set(collection, id, tenantId, ownerId string, obj any, userId string) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	GenCode(tag string) (string, error)
	Query(query et.Json) (et.Items, error)
}

type Tenant struct {
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	ID            string          `json:"id"`
	Tag           string          `json:"tag"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	ServerDB      ServerDB        `json:"server_db"`
	ServerStorage ServerStorage   `json:"server_storage"`
	LimitDB       int             `json:"limit_db"`
	LimitStorage  int             `json:"limit_storage"`
	Apps          map[string]*App `json:"apps"`
	AuditLog      []et.Json       `json:"audit_log"`
	store         Store           `json:"-"`
	isDebug       bool            `json:"-"`
}

/**
* NewTenant
* @param tag, name string, store Store
* @return *Tenant
**/
func NewTenant(tag, name string, store Store) *Tenant {
	now := timezone.Now()
	id := reg.ULID()
	result := &Tenant{
		CreatedAt:    now,
		UpdatedAt:    now,
		ID:           id,
		Tag:          tag,
		Name:         name,
		LimitDB:      10,
		LimitStorage: 100,
		Apps:         make(map[string]*App, 0),
		AuditLog:     make([]et.Json, 0),
		store:        store,
		isDebug:      false,
	}
	return result
}

/**
* LoadTenant
* @param id string, store Store
* @return *Tenant, error
**/
func LoadTenant(id string, store Store) (*Tenant, error) {
	var tenant *Tenant
	exists, err := store.Get("tenant", id, &tenant)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_TENANT_NOT_FOUND)
	}

	tenant.store = store
	return tenant, nil
}

func (s *Tenant) ToJson() et.Json {
	return et.Json{
		"created_at":    s.CreatedAt,
		"updated_at":    s.UpdatedAt,
		"id":            s.ID,
		"tag":           s.Tag,
		"name":          s.Name,
		"description":   s.Description,
		"limit_db":      s.LimitDB,
		"limit_storage": s.LimitStorage,
		"apps":          s.Apps,
		"audit_log":     s.AuditLog,
	}
}

func (s *Tenant) save(userId string) (et.Items, error) {
	s.UpdatedAt = timezone.Now()
	data := s.ToJson()
	data.Set("user_id", userId)
	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	event.Publish(EVENT_TENANT_SET, data)

	return s.store.Query(query)
}
