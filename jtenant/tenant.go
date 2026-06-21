package jtenant

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/config"
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
	CreatedAt     time.Time                                   `json:"created_at"`
	UpdatedAt     time.Time                                   `json:"updated_at"`
	ID            string                                      `json:"id"`
	Tag           string                                      `json:"tag"`
	Name          string                                      `json:"name"`
	Description   string                                      `json:"description"`
	ServerDB      ServerDB                                    `json:"server_db"`
	ServerStorage ServerStorage                               `json:"server_storage"`
	LimitDB       int                                         `json:"limit_db"`
	LimitStorage  int                                         `json:"limit_storage"`
	Apps          map[string]*App                             `json:"apps"`
	AuditLog      []et.Json                                   `json:"audit_log"`
	store         Store                                       `json:"-"`
	isDebug       bool                                        `json:"-"`
	isChanged     bool                                        `json:"-"`
	onSave        []func(tenant *Tenant, userId string) error `json:"-"`
	onDelete      []func(tenant *Tenant, userId string) error `json:"-"`
}

/**
* NewTenant
* @param tag, name string, store Store, userId string
* @return *Tenant
**/
func NewTenant(tag, name string, store Store, userId string) *Tenant {
	now := timezone.Now()
	result := &Tenant{
		CreatedAt:    now,
		UpdatedAt:    now,
		ID:           reg.ULID(),
		Tag:          tag,
		Name:         name,
		LimitDB:      10,
		LimitStorage: 100,
		Apps:         make(map[string]*App, 0),
		AuditLog:     make([]et.Json, 0),
	}
	result.addAuditLog(userId, "new_tenant")
	result.up(store)
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

/**
* ToJson
* @return et.Json
**/
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

/**
* ToString
* @return string
**/
func (s *Tenant) ToString() string {
	return s.ToJson().ToString()
}

/**
* up
* @param store Store
* @return *Tenant
**/
func (s *Tenant) up(store Store) *Tenant {
	s.store = store
	s.isDebug = config.GetBool("DEBUG", false)
	s.onSave = make([]func(tenant *Tenant, userId string) error, 0)
	s.onDelete = make([]func(tenant *Tenant, userId string) error, 0)
	s.OnSave(func(tenant *Tenant, userId string) error {
		key := fmt.Sprintf("tenant:%s", tenant.ID)
		event.Publish(key, tenant.ToJson())
		return nil
	}).OnDelete(func(tenant *Tenant, userId string) error {
		key := fmt.Sprintf("tenant:%s:delete", tenant.ID)
		event.Publish(key, et.Json{
			"id": tenant.ID,
		})
		return nil
	})
	return s
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *Tenant) addAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := config.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
}

/**
* OnSave
* @param fn func(tenant *Tenant, userId string) error
* @return *Tenant
**/
func (s *Tenant) OnSave(fn func(tenant *Tenant, userId string) error) *Tenant {
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(tenant *Tenant, userId string) error
* @return *Tenant
**/
func (s *Tenant) OnDelete(fn func(tenant *Tenant, userId string) error) *Tenant {
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @param userId string
* @return error
**/
func (s *Tenant) save(userId string) error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	s.isChanged = false

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	err := s.store.Set("tenant", s.ID, s.ID, s.ID, s, userId)
	if err != nil {
		return err
	}

	for _, onSave := range s.onSave {
		err := onSave(s, userId)
		if err != nil {
			return err
		}
	}

	return nil
}
