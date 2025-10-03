package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/claim"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/reg"
	lg "github.com/cgalvisleon/et/stdrout"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

var (
	hostName, _  = os.Hostname()
	commonHeader = make(map[string]bool)
	serviceName  = "telemetry"
)

const (
	TELEMETRY                                 = "telemetry"
	TELEMETRY_LOG                             = "telemetry:log"
	TELEMETRY_OVERFLOW                        = "telemetry:overflow"
	TELEMETRY_TOKEN_LAST_USE                  = "telemetry:token:last_use"
	MetricKey                claim.ContextKey = "metric"
)

type Result struct {
	Ok     bool        `json:"ok"`
	Result interface{} `json:"result"`
}

type ResponseWriterWrapper struct {
	http.ResponseWriter
	Size       int
	StatusCode int
}

/**
* Write
* @params b []byte
**/
func (rw *ResponseWriterWrapper) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.Size += size
	return size, err
}

/**
* SetServiceName
* @params name string
**/
func SetServiceName(name string) {
	serviceName = name
}

type Metrics struct {
	TimeStamp    time.Time     `json:"timestamp"`
	ServiceName  string        `json:"service_name"`
	ServiceId    string        `json:"service_id"`
	RemoteAddr   string        `json:"remote_addr"`
	Scheme       string        `json:"scheme"`
	Host         string        `json:"host"`
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	StatusCode   int           `json:"status_code"`
	ResponseSize int           `json:"response_size"`
	SearchTime   time.Duration `json:"search_time"`
	ResponseTime time.Duration `json:"response_time"`
	Latency      time.Duration `json:"latency"`
	AppName      string        `json:"app_name"`
	key          string
	mark         time.Time
	metrics      Telemetry
}

/**
* ToJson
* @return et.Json
**/
func (m *Metrics) ToJson() et.Json {
	return et.Json{
		"timestamp":     strs.FormatDateTime("02/01/2006 03:04:05 PM", m.TimeStamp),
		"service_id":    m.ServiceId,
		"remote_addr":   m.RemoteAddr,
		"scheme":        m.Scheme,
		"host":          m.Host,
		"method":        m.Method,
		"path":          m.Path,
		"status_code":   m.StatusCode,
		"search_time":   m.SearchTime,
		"response_time": m.ResponseTime,
		"latency":       m.Latency,
		"response_size": m.ResponseSize,
		"app_name":      m.AppName,
	}
}

type Telemetry struct {
	TimeStamp         string
	ServiceName       string
	Key               string
	RequestsPerSecond int64
	RequestsPerMinute int64
	RequestsPerHour   int64
	RequestsPerDay    int64
	RequestsLimit     int64
}

/**
* ToJson
* @return et.Json
**/
func (m *Telemetry) ToJson() et.Json {
	return et.Json{
		"timestamp":           m.TimeStamp,
		"key":                 m.Key,
		"service_name":        m.ServiceName,
		"requests_per_second": m.RequestsPerSecond,
		"requests_per_minute": m.RequestsPerMinute,
		"requests_per_hour":   m.RequestsPerHour,
		"requests_per_day":    m.RequestsPerDay,
		"requests_limit":      m.RequestsLimit,
	}
}

/**
* GetMetrics
* @params r *http.Request
* @return *Metrics
**/
func GetMetrics(r *http.Request) *Metrics {
	metric, ok := r.Context().Value(MetricKey).(*Metrics)
	if !ok {
		return NewMetric(r)
	}

	return metric
}

/**
* PushTelemetry
* @param data et.Json
**/
func PushTelemetry(data et.Json) {
	go event.Publish(TELEMETRY, data)
}

/**
* PushTelemetryLog
* @param data string
**/
func PushTelemetryLog(data string) {
	go event.Publish(TELEMETRY_LOG, et.Json{
		"log": data,
	})
}

/**
* PushTelemetryOverflow
* @param data et.Json
**/
func PushTelemetryOverflow(data et.Json) {
	go event.Publish(TELEMETRY_OVERFLOW, data)
}

/**
* TokenLastUse
* @param data et.Json
**/
func PushTokenLastUse(data et.Json) {
	go event.Publish(TELEMETRY_TOKEN_LAST_USE, data)
}

/**
* NewMetric
* @params r *http.Request
* @return *Metrics
**/
func NewMetric(r *http.Request) *Metrics {
	remoteAddr := r.RemoteAddr
	if remoteAddr == "" {
		remoteAddr = r.Header.Get("X-Forwarded-For")
	}
	if remoteAddr == "" {
		remoteAddr = r.Header.Get("X-Real-IP")
	}
	if remoteAddr != "" {
		remoteAddr = strs.Split(remoteAddr, ",")[0]
	}
	serviceId := r.Header.Get("ServiceId")
	if serviceId == "" {
		serviceId = utility.UUID()
		r.Header.Set("ServiceId", serviceId)
	}
	appName := "Not Found"
	if r.Header.Get("AppName") != "" {
		appName = r.Header.Get("AppName")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	result := &Metrics{
		TimeStamp:   timezone.NowTime(),
		ServiceName: serviceName,
		ServiceId:   serviceId,
		RemoteAddr:  remoteAddr,
		Host:        hostName,
		Method:      r.Method,
		Path:        r.URL.Path,
		Scheme:      scheme,
		AppName:     appName,
		mark:        timezone.NowTime(),
		key:         fmt.Sprintf(`%s:%s`, r.Method, r.URL.Path),
	}

	return result
}

/**
* NewRpcMetric
* @params method string
* @return *Metrics
**/
func NewRpcMetric(method string) *Metrics {
	scheme := "rpc"
	result := &Metrics{
		TimeStamp:   timezone.NowTime(),
		ServiceName: serviceName,
		ServiceId:   utility.UUID(),
		Path:        method,
		Method:      strs.Uppcase(scheme),
		Scheme:      scheme,
		mark:        timezone.NowTime(),
		key:         fmt.Sprintf(`%s:%s`, strs.Uppcase(scheme), method),
	}

	return result
}

/**
* setRequest
* @params remove bool
**/
func (m *Metrics) setRequest(remove bool) {
	m.key = fmt.Sprintf(`%s:%s`, m.Method, m.Path)
	if remove {
		cache.LRem("telemetry:requests", m.key)
	} else {
		cache.LPush("telemetry:requests", m.key)
	}
}

/**
* SetPath
* @params val string
**/
func (m *Metrics) SetPath(val string) {
	if val == "" {
		return
	}

	m.Path = val
	m.setRequest(false)
}

/**
* CallSearchTime
**/
func (m *Metrics) CallSearchTime() {
	m.SearchTime = time.Since(m.mark)
	m.mark = timezone.NowTime()
}

/**
* CallResponseTime
**/
func (m *Metrics) CallResponseTime() {
	m.ResponseTime = time.Since(m.mark)
	m.mark = timezone.NowTime()
}

/**
* CallLatency
**/
func (m *Metrics) CallLatency() {
	m.Latency = time.Since(m.TimeStamp)
}

/**
* CallMetrics
* @return Telemetry
**/
func (m *Metrics) CallMetrics() Telemetry {
	timeNow := timezone.NowTime()
	date := timeNow.Format("2006-01-02")
	hour := timeNow.Format("2006-01-02-15")
	minute := timeNow.Format("2006-01-02-15:04")
	second := timeNow.Format("2006-01-02-15:04:05")

	return Telemetry{
		TimeStamp:         date,
		ServiceName:       serviceName,
		Key:               m.key,
		RequestsPerSecond: cache.Incr(reg.GenHashKey(m.key, second), 2),
		RequestsPerMinute: cache.Incr(reg.GenHashKey(m.key, minute), 60),
		RequestsPerHour:   cache.Incr(reg.GenHashKey(m.key, hour), 3600),
		RequestsPerDay:    cache.Incr(reg.GenHashKey(m.key, date), 86400),
		RequestsLimit:     int64(config.GetInt("REQUESTS_LIMIT", 400)),
	}
}

/**
* println
* @return et.Json
**/
func (m *Metrics) println() et.Json {
	w := lg.Color(lg.NMagenta, " [%s]: ", m.Method)
	lg.CW(w, lg.NCyan, "%s", m.Path)
	lg.CW(w, lg.NWhite, " from:%s", m.RemoteAddr)
	if m.StatusCode >= 500 {
		lg.CW(w, lg.NRed, " - %s", http.StatusText(m.StatusCode))
	} else if m.StatusCode >= 400 {
		lg.CW(w, lg.NYellow, " - %s", http.StatusText(m.StatusCode))
	} else if m.StatusCode >= 300 {
		lg.CW(w, lg.NCyan, " - %s", http.StatusText(m.StatusCode))
	} else {
		lg.CW(w, lg.NGreen, " - %s", http.StatusText(m.StatusCode))
	}
	size := float64(m.ResponseSize) / 1024
	lg.CW(w, lg.NCyan, " Size:%.2f%s", size, "KB")
	lg.CW(w, lg.NWhite, " in ")
	limitLatency := time.Duration(config.GetInt("LATENCY_LIMIT", 1000)) * time.Millisecond
	if m.Latency < limitLatency {
		lg.CW(w, lg.NGreen, " Latency:%s", m.Latency)
	} else if m.Latency < 5*time.Second {
		lg.CW(w, lg.NYellow, " Latency:%s", m.Latency)
	} else {
		lg.CW(w, lg.NRed, " Latency:%s", m.Latency)
	}
	lg.CW(w, lg.NWhite, " Response:%s", m.ResponseTime)
	m.metrics = m.CallMetrics()
	requestsLimit := float64(m.metrics.RequestsLimit) * 0.6
	if m.metrics.RequestsPerSecond > m.metrics.RequestsLimit {
		lg.CW(w, lg.NRed, " - Request:S:%vM:%vH:%vD:%vL:%v", m.metrics.RequestsPerSecond, m.metrics.RequestsPerMinute, m.metrics.RequestsPerHour, m.metrics.RequestsPerDay, m.metrics.RequestsLimit)
	} else if m.metrics.RequestsPerSecond > int64(requestsLimit) {
		lg.CW(w, lg.NYellow, " - Request:S:%vM:%vH:%vD:%vL:%v", m.metrics.RequestsPerSecond, m.metrics.RequestsPerMinute, m.metrics.RequestsPerHour, m.metrics.RequestsPerDay, m.metrics.RequestsLimit)
	} else {
		lg.CW(w, lg.NGreen, " - Request:S:%vM:%vH:%vD:%vL:%v", m.metrics.RequestsPerSecond, m.metrics.RequestsPerMinute, m.metrics.RequestsPerHour, m.metrics.RequestsPerDay, m.metrics.RequestsLimit)
	}
	lg.CW(w, lg.NCyan, " [ServiceId]:%s", m.ServiceId)
	lg.CW(w, lg.NCyan, " [AppName]:%s", m.AppName)
	lg.Println(w)

	m.setRequest(true)
	PushTelemetryLog(w.String())

	return m.ToJson()
}

/**
* telemetry
* @return et.Json
**/
func (m *Metrics) telemetry() et.Json {
	result := m.ToJson()
	result["metric"] = m.metrics.ToJson()

	PushTelemetry(et.Json{
		"response": m,
		"metric":   m.metrics.ToJson(),
	})

	if m.metrics.RequestsPerSecond > m.metrics.RequestsLimit {
		PushTelemetryOverflow(et.Json{
			"response": m,
			"metric":   m.metrics.ToJson(),
		})
	}

	return result
}

/**
* DoneHTTP
* @params rw *ResponseWriterWrapper
* @return et.Json
**/
func (m *Metrics) DoneHTTP(rw *ResponseWriterWrapper) et.Json {
	m.StatusCode = rw.StatusCode
	m.ResponseSize = rw.Size
	m.CallResponseTime()
	m.CallLatency()
	m.println()

	return m.telemetry()
}

/**
* DoneRpc
* @params r et.Json
* @return et.Json
**/
func (m *Metrics) DoneRpc(r any) et.Json {
	switch v := r.(type) {
	case string:
		m.ResponseSize = len(v)
	case et.Json:
		m.ResponseSize = len(v.ToString())
	case []byte:
		m.ResponseSize = len(v)
	case int:
		m.ResponseSize = len(strconv.Itoa(v))
	case float64:
		m.ResponseSize = len(strconv.FormatFloat(v, 'f', -1, 64))
	case bool:
		m.ResponseSize = len(strconv.FormatBool(v))
	case et.List:
		m.ResponseSize = len(v.ToString())
	case et.Items:
		m.ResponseSize = len(v.ToString())
	case et.Item:
		m.ResponseSize = len(v.ToString())
	default:
		m.ResponseSize = len(fmt.Sprintf("%v", v))
	}

	m.StatusCode = http.StatusOK
	m.CallResponseTime()
	m.CallLatency()
	m.println()

	return m.telemetry()
}

/**
* WriteResponse
* @params w http.ResponseWriter, r *http.Request, statusCode int, e []byte
* @return error
**/
func (m *Metrics) WriteResponse(w http.ResponseWriter, r *http.Request, statusCode int, e []byte) error {
	rw := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: statusCode}

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(statusCode)
	rw.Write(e)

	m.DoneHTTP(rw)
	return nil
}

/**
* RESULT
* @param w http.ResponseWriter, r *http.Request, statusCode int, data interface{}
* @return error
**/
func (m *Metrics) RESULT(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) error {
	if data == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return m.WriteResponse(w, r, statusCode, e)
}

/**
* JSON
* @params w http.ResponseWriter, r *http.Request, statusCode int, data interface{}
* @return error
**/
func (m *Metrics) JSON(w http.ResponseWriter, r *http.Request, statusCode int, dt interface{}) error {
	if dt == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	result := Result{
		Ok:     http.StatusOK == statusCode,
		Result: dt,
	}

	e, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return m.WriteResponse(w, r, statusCode, e)
}

/**
* ITEM
* @params w http.ResponseWriter, r *http.Request, statusCode int, dt et.Item
* @return error
**/
func (m *Metrics) ITEM(w http.ResponseWriter, r *http.Request, statusCode int, dt et.Item) error {
	if &dt == (&et.Item{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(dt)
	if err != nil {
		return err
	}

	return m.WriteResponse(w, r, statusCode, e)
}

/**
* ITEMS
* @params w http.ResponseWriter, r *http.Request, statusCode int, dt et.Items
* @return error
**/
func (m *Metrics) ITEMS(w http.ResponseWriter, r *http.Request, statusCode int, dt et.Items) error {
	if &dt == (&et.Items{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}

	e, err := json.Marshal(dt)
	if err != nil {
		return err
	}

	return m.WriteResponse(w, r, statusCode, e)
}

/**
* HTTPError
* @params w http.ResponseWriter, r *http.Request, statusCode int, message string
* @return error
**/
func (m *Metrics) HTTPError(w http.ResponseWriter, r *http.Request, statusCode int, message string) error {
	msg := et.Json{
		"message": message,
	}

	return m.JSON(w, r, statusCode, msg)
}

/**
* Unauthorized
* @params w http.ResponseWriter, r *http.Request
**/
func (m *Metrics) Unauthorized(w http.ResponseWriter, r *http.Request) {
	m.HTTPError(w, r, http.StatusUnauthorized, "401 Unauthorized")
}

func init() {
	for _, v := range []string{
		"Content-Security-Policy",
		"Content-Length",
	} {
		commonHeader[v] = true
	}
}
