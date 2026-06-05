package cache

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Metrics struct {
	TimeStamp         string `json:"timestamp"`
	Key               string `json:"key"`
	RequestsPerSecond int64  `json:"requests_per_second"`
	RequestsPerMinute int64  `json:"requests_per_minute"`
	RequestsPerHour   int64  `json:"requests_per_hour"`
	RequestsPerDay    int64  `json:"requests_per_day"`
	RequestsLimit     int64  `json:"requests_limit"`
	OverLimit         bool   `json:"over_limit"`
}

/**
* ToJson
* @return et.Json
**/
func (s *Metrics) ToJson() et.Json {
	return et.Json{
		"timestamp":           s.TimeStamp,
		"key":                 s.Key,
		"requests_per_second": s.RequestsPerSecond,
		"requests_per_minute": s.RequestsPerMinute,
		"requests_per_hour":   s.RequestsPerHour,
		"requests_per_day":    s.RequestsPerDay,
		"requests_limit":      s.RequestsLimit,
		"over_limit":          s.OverLimit,
	}
}

/**
* CallMetrics
* @params key string, limit int64
* @return Metrics
**/
func CallMetrics(key string, limit int64) (Metrics, error) {
	if conn == nil {
		return Metrics{}, errors.New(msg.MSG_NOT_CACHE_SERVICE)
	}

	timeNow := timezone.Now()
	date := timeNow.Format("2006-01-02")
	hour := timeNow.Format("2006-01-02-15")
	minute := timeNow.Format("2006-01-02-15:04")
	second := timeNow.Format("2006-01-02-15:04:05")

	result := Metrics{
		TimeStamp:         date,
		Key:               key,
		RequestsPerSecond: Incr(reg.GenHashKey(key, second), 2*time.Second),
		RequestsPerMinute: Incr(reg.GenHashKey(key, minute), time.Minute),
		RequestsPerHour:   Incr(reg.GenHashKey(key, hour), time.Hour),
		RequestsPerDay:    Incr(reg.GenHashKey(key, date), 24*time.Hour),
		RequestsLimit:     limit,
	}
	result.OverLimit = result.RequestsPerSecond > result.RequestsLimit
	return result, nil
}
