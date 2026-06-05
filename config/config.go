package config

import (
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
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
	Params    et.Json   `json:"params"`
	Stage     string    `json:"stage"`
	store     Store     `json:"-"`
	isDebug   bool      `json:"-"`
}

var (
	packageName = "config"
	CNF         *Config
)

/**
* Load
* @param tag, stage, tenantId, ownerId string, store Store
* @return error
**/
func Load(tag, stage, tenantId, ownerId string, store Store, userId string) error {
	var err error
	CNF, err = New(tag, stage, tenantId, ownerId, store, userId)
	if err != nil {
		return err
	}

	return nil
}

/**
* NewConfig
* @param tag, stage, tenantId, ownerId string, store Store, userId string
* @return *Config
**/
func New(tag, stage, tenantId, ownerId string, store Store, userId string) (*Config, error) {
	if utility.ValidStr(tag, 1, []string{""}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "tag")
	}

	if utility.ValidStr(stage, 1, []string{""}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "stage")
	}

	if utility.ValidStr(tenantId, 1, []string{""}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "tenantId")
	}

	new := func() (*Config, error) {
		now := timezone.Now()
		id := fmt.Sprintf("config:%s:%s:%s", tag, stage, tenantId)
		result := &Config{
			CreatedAt: now,
			UpdatedAt: now,
			ID:        id,
			Params:    et.Json{},
			Tag:       tag,
			Stage:     stage,
			OwnerId:   ownerId,
			TenantId:  tenantId,
			store:     store,
		}
		if store != nil {
			err := result.save(userId)
			if err != nil {
				return nil, err
			}
		}

		return result, nil
	}

	if store != nil {
		var result *Config
		exists, err := store.Get(tag, stage, result)
		if err != nil {
			return nil, err
		}

		if !exists {
			return new()
		}

		bt, err := json.Marshal(CNF)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(bt, &result)
		if err != nil {
			return nil, err
		}

		return CNF, nil
	}

	return new()
}

/**
* Serialize
* @return ([]byte, error)
**/
func (s *Config) Serialize() ([]byte, error) {
	return utility.Serialize(s)
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
* save
* @param userId string
* @return error
**/
func (s *Config) save(userId string) error {
	s.UpdatedAt = timezone.Now()
	data := s.ToJson()
	data.Set("user_id", userId)
	if s.isDebug {
		logs.Logf(packageName, "save: %s", data.ToString())
	}

	if s.store != nil {
		return s.store.Set(s.Tag, s.Stage, s.OwnerId, s.TenantId, s.Params, userId)
	}

	event.Publish(EVENT_CONFIG_SET, data)

	return nil
}

/**
* Set
* @param key string, value interface{}
* @return error
**/
func (s *Config) Set(param et.Json, userId string) error {
	maps.Copy(s.Params, param)
	return s.save(userId)
}

/**
* Delete
* @param key string, userId string
* @return error
**/
func (s *Config) Delete(key string, userId string) error {
	delete(s.Params, key)
	return s.save(userId)
}

/**
* Get
* @param key string, def interface{}
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
