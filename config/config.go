package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/envar"
)

type Store interface {
	Get(tag, stage string, dest any) (bool, error)
	Set(tag, stage, tenantId, ownerId string, obj any, userId string) error
	Delete(tag, stage string) error
}

type Config struct {
	ID       string                   `json:"id"`
	TenantId string                   `json:"tenant_id"`
	OwnerId  string                   `json:"owner_id"`
	Tag      string                   `json:"tag"`
	Stage    string                   `json:"stage"`
	Params   map[string]interface{}   `json:"params"`
	AuditLog []map[string]interface{} `json:"audit_log"`
	store    Store                    `json:"-"`
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

	id := fmt.Sprintf("config:%s:%s:%s", tag, stage, tenantId)
	result := &Config{
		TenantId: tenantId,
		OwnerId:  ownerId,
		ID:       id,
		Tag:      tag,
		Stage:    stage,
		Params:   map[string]interface{}{},
		AuditLog: make([]map[string]interface{}, 0),
		store:    store,
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
func (s *Config) Save(updateAt time.Time, userId string) error {
	if s.store == nil {
		return errors.New(MSG_CONFIG_STORE_IS_NIL)
	}

	s.AuditLog = append(s.AuditLog, map[string]interface{}{
		"created_at": updateAt,
		"user_id":    userId,
		"action":     "save",
	})
	maxAuditLog := GetInt("MAX_AUDIT_LOG", 1000)
	s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]

	return s.store.Set(s.Tag, s.Stage, s.OwnerId, s.TenantId, s, userId)
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
