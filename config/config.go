package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
)

type Store interface {
	Get(tag, stage string, dest any) (bool, error)
	Set(tag, stage, tenantId, ownerId string, obj any, userId string) error
	Delete(tag, stage string) error
	Query(query et.Json) (et.Items, error)
}

type Config struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ID        string    `json:"id"`
	TenantId  string    `json:"tenant_id"`
	OwnerId   string    `json:"owner_id"`
	Tag       string    `json:"tag"`
	Stage     string    `json:"stage"`
	Params    et.Json   `json:"params"`
	AuditLog  []et.Json `json:"audit_log"`
	store     Store     `json:"-"`
	isDebug   bool      `json:"-"`
}

var (
	packageName = "config"
	cnf         *Config
)

/**
* NewConfig
* @param tag, stage, tenantId, ownerId string, store Store, userId string
* @return *Config
**/
func New(tag, stage, tenantId, ownerId string, store Store, userId string) (*Config, error) {
	if tag == "" {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "tag")
	}

	if stage == "" {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "stage")
	}

	if tenantId == "" {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "tenantId")
	}

	now := timezone.Now()
	id := fmt.Sprintf("config:%s:%s:%s", tag, stage, tenantId)
	result := &Config{
		CreatedAt: now,
		UpdatedAt: now,
		TenantId:  tenantId,
		OwnerId:   ownerId,
		ID:        id,
		Tag:       tag,
		Stage:     stage,
		Params:    et.Json{},
		AuditLog:  make([]et.Json, 0),
		store:     store,
	}
	return result, nil
}

/**
* Load
* @param tag, stage, tenantId, ownerId string, store Store
* @return error
**/
func Load(tag, stage, tenantId, ownerId string, store Store, userId string) error {
	if store == nil {
		return errors.New(MSG_CONFIG_STORE_IS_NIL)
	}

	exists, err := store.Get(tag, stage, cnf)
	if err != nil {
		return err
	}

	if !exists {
		cnf, err = New(tag, stage, tenantId, ownerId, store, userId)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* Save
* @param userId string
* @return error
**/
func (s *Config) Save(userId string) error {
	if s.store == nil {
		return errors.New(MSG_CONFIG_STORE_IS_NIL)
	}

	now := timezone.Now()
	s.UpdatedAt = now
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     "save",
	})
	maxAuditLog := GetInt("MAX_AUDIT_LOG", 1000)
	s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]

	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	return s.store.Set(s.Tag, s.Stage, s.OwnerId, s.TenantId, s, userId)
}

/**
* ToJson
* @return et.Json
**/
func (s *Config) ToJson() et.Json {
	return et.Json{
		"created_at": timezone.Format(s.CreatedAt, timezone.RFC3339),
		"updated_at": timezone.Format(s.UpdatedAt, timezone.RFC3339),
		"id":         s.ID,
		"tenant_id":  s.TenantId,
		"owner_id":   s.OwnerId,
		"tag":        s.Tag,
		"params":     s.Params,
		"stage":      s.Stage,
	}
}

/**
* ToString
* @return string
**/
func (s *Config) ToString() string {
	return s.ToJson().ToString()
}

/**
* Debug
* @return *Config
**/
func (s *Config) Debug() *Config {
	s.isDebug = true
	return s
}

/**
* Set
* @param param et.Json
* @return *Config
**/
func (s *Config) Set(param et.Json) *Config {
	maps.Copy(s.Params, param)
	return s
}

/**
* Delete
* @param key string
* @return *Config
**/
func (s *Config) Remove(key string) *Config {
	delete(s.Params, key)
	return s
}

/**
* Get
* @param key string
* @return interface{}
**/
func (s *Config) Get(key string, def interface{}) interface{} {
	result, ok := s.Params[key]
	if ok {
		return result
	}
	return envar.Get(key, def)
}

/**
* GetStr
* @param key string, def string
* @return string
**/
func (s *Config) GetStr(key string, def string) string {
	result := s.Get(key, def)
	return fmt.Sprintf("%v", result)
}

/**
* GetInt
* @param key string, def int
* @return int
**/
func (s *Config) GetInt(key string, def int) int {
	result := s.GetStr(key, strconv.Itoa(def))
	val, err := strconv.Atoi(result)
	if err != nil {
		return def
	}

	return val
}

/**
* GetInt64
* @param key string, def int64
* @return int64
**/
func (s *Config) GetInt64(key string, def int64) int64 {
	result := s.GetStr(key, strconv.FormatInt(def, 10))
	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return def
	}

	return val
}

/**
* GetFloat
* @param key string, def float64
* @return float64
**/
func (s *Config) GetFloat(key string, def float64) float64 {
	result := s.GetStr(key, strconv.FormatFloat(def, 'f', -1, 64))
	val, err := strconv.ParseFloat(result, 64)
	if err != nil {
		return def
	}
	return val
}

/**
* GetBool
* @param key string, def bool
* @return bool
**/
func (s *Config) GetBool(key string, def bool) bool {
	result := s.GetStr(key, strconv.FormatBool(def))
	val, err := strconv.ParseBool(result)
	if err != nil {
		return def
	}
	return val
}

/**
* GetTime
* @param key string, def time.Time
* @return time.Time
**/
func (s *Config) GetTime(key string, def time.Time) time.Time {
	result := s.GetStr(key, timezone.Format(def, timezone.RFC3339))
	val, err := timezone.Parse(timezone.RFC3339, result)
	if err != nil {
		return def
	}
	return val
}

/**
* GetJson
* @param key string, def et.Json
* @return et.Json
**/
func (s *Config) GetJson(key string, def et.Json) et.Json {
	result := s.GetStr(key, def.ToString())
	var resultJson et.Json
	err := json.Unmarshal([]byte(result), &resultJson)
	if err != nil {
		return def
	}
	return resultJson
}
