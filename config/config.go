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
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Store interface {
	Set(tag, stage, tenantId, ownerId string, obj any) error
	Get(tag, stage string, dest any) (bool, error)
	Delete(tag, stage string) error
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
	isDebug   bool      `json:"-"`
	isChanged bool      `json:"-"`
	store     Store     `json:"-"`
}

var (
	cnf         *Config
	packageName = "config"
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

	result := &Config{
		TenantId: tenantId,
		OwnerId:  ownerId,
		ID:       reg.ULID(),
		Tag:      tag,
		Stage:    stage,
		Params:   et.Json{},
		AuditLog: make([]et.Json, 0),
		store:    store,
	}
	result.addAuditLog(userId, "new_config")
	return result, nil
}

/**
* Load
* @param tag, stage string, store Store, userId string
* @return error
**/
func Load(tag, stage string, store Store, userId string) error {
	if store == nil {
		return errors.New(MSG_CONFIG_STORE_IS_NIL)
	}

	exists, err := store.Get(tag, stage, cnf)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New(MSG_CONFIG_NOT_LOADED)
	}

	return nil
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *Config) addAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.UpdatedAt = now
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
}

/**
* Save
* @param userId string
* @return error
**/
func (s *Config) Save() error {
	if s.store == nil {
		return errors.New(MSG_CONFIG_STORE_IS_NIL)
	}

	s.isChanged = false
	if s.isDebug {
		logs.Log(packageName, "save:", s.ToString())
	}

	return s.store.Set(s.Tag, s.Stage, s.OwnerId, s.TenantId, s)
}

/**
* ToJson
* @return map[string]interface{}
**/
func (s *Config) ToJson() map[string]interface{} {
	return map[string]interface{}{
		"id":        s.ID,
		"tenant_id": s.TenantId,
		"owner_id":  s.OwnerId,
		"tag":       s.Tag,
		"params":    s.Params,
		"stage":     s.Stage,
	}
}

/**
* ToString
* @return string
**/
func (s *Config) ToString() string {
	bt, err := json.Marshal(s)
	if err != nil {
		return ""
	}

	return string(bt)
}

/**
* Set
* @param param map[string]interface{}
* @return *Config
**/
func (s *Config) Set(param map[string]interface{}) *Config {
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
* GetMap
* @param key string, def map[string]any
* @return map[string]any
**/
func (s *Config) GetMap(key string, def map[string]any) map[string]any {
	str := s.GetStr(key, "{}")
	var result map[string]any
	err := json.Unmarshal([]byte(str), &result)
	if err != nil {
		return def
	}
	return result
}
