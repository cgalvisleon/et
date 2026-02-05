package mem

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

type Entry struct {
	Key        string
	Value      []byte
	Version    int
	LastUpdate time.Time
	Expiration time.Duration
}

/**
* New create new item
* @param key string
* @param value interface{}
* @return *Entry
**/
func New(key string, value interface{}, expiration time.Duration) (*Entry, error) {
	now := timezone.Now()
	bt, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	return &Entry{
		LastUpdate: now,
		Expiration: expiration,
		Key:        key,
		Value:      bt,
	}, nil
}

/**
* Set a value in item
* @param value interface{}, expiration time.Duration
* @return *Entry, error
**/
func (s *Entry) Set(value interface{}, expiration time.Duration) (*Entry, error) {
	bt, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	s.LastUpdate = timezone.Now()
	s.Expiration = expiration
	s.Value = bt
	s.Version++

	return s, nil
}

/**
* Get a value from item
* @return []byte
**/
func (s *Entry) Get() []byte {
	return s.Value
}

/**
* Str return the value of item
* @return string
**/
func (s *Entry) Str() string {
	result := s.Get()
	return string(result)
}

/**
* Int return the value of item
* @return int, error
**/
func (s *Entry) Int() (int, error) {
	var result int
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* Int64 return the value of item
* @return int64, error
**/
func (s *Entry) Int64() (int64, error) {
	var result int64
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* Float return the value of item
* @return float64, error
**/
func (s *Entry) Float() (float64, error) {
	var result float64
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

/**
* Bool return the value of item
* @return bool, error
**/
func (s *Entry) Bool() (bool, error) {
	var result bool
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return false, err
	}

	return result, nil
}

/**
* Time return the value of item
* @return time.Time, error
**/
func (s *Entry) Time() (time.Time, error) {
	var result time.Time
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return time.Time{}, err
	}

	return result, nil
}

/**
* Duration return the value of item
* @return time.Duration, error
**/
func (s *Entry) Duration() (time.Duration, error) {
	var result time.Duration
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return time.Duration(0), err
	}

	return result, nil
}

/**
* Json return the value of item
* @return et.Json, error
**/
func (s *Entry) Json() (et.Json, error) {
	var result et.Json
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}, err
	}

	return result, nil
}

/**
* Map return the value of item
* @return map[string]interface{}, error
**/
func (s *Entry) Map() (map[string]interface{}, error) {
	var result map[string]interface{}
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return map[string]interface{}{}, err
	}

	return result, nil
}

/**
* ArrayMap return the value of item
* @return []interface{}, error
**/
func (s *Entry) ArrayMap() ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return []map[string]interface{}{}, err
	}

	return result, nil
}

/**
* ArrayStr return the value of item
* @return []string, error
**/
func (s *Entry) ArrayStr() ([]string, error) {
	var result []string
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return []string{}, err
	}

	return result, nil
}

/**
* ArrayInt return the value of item
* @return []int, error
**/
func (s *Entry) ArrayInt() ([]int, error) {
	var result []int
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return []int{}, err
	}

	return result, nil
}

/**
* ArrayFloat return the value of item
* @return []float64, error
**/
func (s *Entry) ArrayFloat() ([]float64, error) {
	var result []float64
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return []float64{}, err
	}

	return result, nil
}

/**
* ArrayTime return the value of item
* @return []time.Time, error
**/
func (s *Entry) ArrayTime() ([]time.Time, error) {
	var result []time.Time
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return []time.Time{}, err
	}

	return result, nil
}

/**
* ArrayDuration return the value of item
* @return []time.Duration, error
**/
func (s *Entry) ArrayDuration() ([]time.Duration, error) {
	var result []time.Duration
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return []time.Duration{}, err
	}

	return result, nil
}

/**
* ArrayJson return the value of item
* @return []et.Json, error
**/
func (s *Entry) ArrayJson() ([]et.Json, error) {
	var result []et.Json
	bt := s.Get()
	err := json.Unmarshal(bt, &result)
	if err != nil {
		return []et.Json{}, err
	}

	return result, nil
}
