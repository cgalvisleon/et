package jtenant

import (
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/jwf"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type App struct {
	CreatedAt   time.Time                             `json:"created_at"`
	UpdatedAt   time.Time                             `json:"updated_at"`
	TenantID    string                                `json:"tenant_id"`
	ID          string                                `json:"id"`
	Tag         string                                `json:"tag"`
	Name        string                                `json:"name"`
	Description string                                `json:"description"`
	DBS         map[string]*jsql.DB                   `json:"dbs"`
	WorkFlows   map[string]*jwf.WorkFlow              `json:"workflows"`
	AuditLog    []et.Json                             `json:"audit_log"`
	store       Store                                 `json:"-"`
	isDebug     bool                                  `json:"-"`
	isChanged   bool                                  `json:"-"`
	onSave      []func(app *App, userId string) error `json:"-"`
	onDelete    []func(app *App, userId string) error `json:"-"`
}

/**
* NewApp
* @param tag, name string, store Store, userId string
* @return *App
**/
func (s *Tenant) NewApp(tag, name string, store Store, userId string) *App {
	now := timezone.Now()
	result := &App{
		CreatedAt:   now,
		UpdatedAt:   now,
		TenantID:    s.ID,
		ID:          reg.ULID(),
		Tag:         tag,
		Name:        name,
		Description: "",
		AuditLog:    make([]et.Json, 0),
	}
	result.addAuditLog(userId, "new_app")
	return result.up(s)
}

func (s *App) up(tenant *Tenant) *App {
	s.store = tenant.store
	s.isDebug = tenant.isDebug
	s.onSave = make([]func(app *App, userId string) error, 0)
	s.onDelete = make([]func(app *App, userId string) error, 0)
	s.
		OnSave(func(app *App, userId string) error {
			key := fmt.Sprintf("app:%s", app.ID)
			event.Publish(key, app.ToJson())
			return nil
		}).
		OnDelete(func(app *App, userId string) error {
			key := fmt.Sprintf("app:%s:delete", app.ID)
			event.Publish(key, et.Json{
				"id": app.ID,
			})
			return nil
		})
	return s
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *App) addAuditLog(userId string, action string) {
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
* @param fn func(app *App, userId string) error
* @return *App
**/
func (s *App) OnSave(fn func(app *App, userId string) error) *App {
	s.onSave = append(s.onSave, fn)
	return s
}

/**
* OnDelete
* @param fn func(app *App, userId string) error
* @return *App
**/
func (s *App) OnDelete(fn func(app *App, userId string) error) *App {
	s.onDelete = append(s.onDelete, fn)
	return s
}

/**
* save
* @param userId string
* @return error
**/
func (s *App) save(userId string) error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	s.isChanged = false
	data := s.ToJson()

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	err := s.store.Set("app", s.ID, s.TenantID, s.ID, s, userId)
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

/**
* ToJson
* @return et.Json
**/
func (s *App) ToJson() et.Json {
	dbs := et.Json{}
	for _, db := range s.DBS {
		dbs[db.ID] = db.ToJson()
	}
	workflows := et.Json{}
	for _, workflow := range s.WorkFlows {
		workflows[workflow.ID] = workflow.ToJson()
	}
	return et.Json{
		"created_at":  timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at":  timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"tenant_id":   s.TenantID,
		"id":          s.ID,
		"tag":         s.Tag,
		"name":        s.Name,
		"description": s.Description,
		"dbs":         dbs,
		"workflows":   workflows,
		"audit_log":   s.AuditLog,
	}
}

/**
* ToString
* @return string
**/
func (s *App) ToString() string {
	return s.ToJson().ToString()
}
