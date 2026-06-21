package jtenant

import (
	"time"

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
