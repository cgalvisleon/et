package jtenant

import (
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/jwf"
)

type App struct {
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
	TenantID    string                   `json:"tenant_id"`
	ID          string                   `json:"id"`
	Tag         string                   `json:"tag"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	DBS         map[string]*jsql.DB      `json:"dbs"`
	WorkFlows   map[string]*jwf.WorkFlow `json:"workflows"`
	store       Store                    `json:"-"`
}

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
	Apps          map[string]*App `json:"apps"`
	AuditLog      []et.Json       `json:"audit_log"`
	store         Store           `json:"-"`
	isDebug       bool            `json:"-"`
}
