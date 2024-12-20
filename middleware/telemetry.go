package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	lg "github.com/cgalvisleon/et/stdrout"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

var hostName, _ = os.Hostname()
var commonHeader = make(map[string]bool)
var serviceName = "telemetry"

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
	ReqID        string        `json:"req_id"`
	ClientIP     string        `json:"client_ip"`
	Scheme       string        `json:"scheme"`
	Host         string        `json:"host"`
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	StatusCode   int           `json:"status_code"`
	ResponseSize int           `json:"response_size"`
	SearchTime   time.Duration `json:"search_time"`
	ResponseTime time.Duration `json:"response_time"`
	Latency      time.Duration `json:"latency"`
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
		"req_id":        m.ReqID,
		"client_ip":     m.ClientIP,
		"scheme":        m.Scheme,
		"host":          m.Host,
		"method":        m.Method,
		"path":          m.Path,
		"status_code":   m.StatusCode,
		"search_time":   m.SearchTime,
		"response_time": m.ResponseTime,
		"latency":       m.Latency,
		"response_size": m.ResponseSize,
	}
}

type Telemetry struct {
	TimeStamp         string
	ServiceName       string
	Key               string
	RequestsPerSecond int
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
	RequestsLimit     int
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
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	result := &Metrics{
		TimeStamp:   timezone.NowTime(),
		ServiceName: serviceName,
		ReqID:       utility.UUID(),
		ClientIP:    remoteAddr,
		Host:        hostName,
		Method:      r.Method,
		Path:        r.URL.Path,
		Scheme:      scheme,
		mark:        timezone.NowTime(),
		key:         strs.Format(`%s:%s`, r.Method, r.URL.Path),
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
		ReqID:       utility.UUID(),
		Path:        method,
		Method:      strs.Uppcase(scheme),
		Scheme:      scheme,
		mark:        timezone.NowTime(),
		key:         strs.Format(`%s:%s`, strs.Uppcase(scheme), method),
	}

	return result
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
	m.key = strs.Format(`%s:%s`, m.Method, m.Path)
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
		RequestsPerSecond: cache.Count(cache.GenKey(m.key, second), 2*time.Second),
		RequestsPerMinute: cache.Count(cache.GenKey(m.key, minute), 1*time.Minute+1*time.Second),
		RequestsPerHour:   cache.Count(cache.GenKey(m.key, hour), 1*time.Hour+1*time.Second),
		RequestsPerDay:    cache.Count(cache.GenKey(m.key, date), 24*time.Hour+1*time.Second),
		RequestsLimit:     envar.GetInt(400, "LIMIT_REQUESTS"),
	}
}

/**
* println
* @return et.Json
**/
func (m *Metrics) println() et.Json {
	w := lg.Color(lg.NMagenta, fmt.Sprintf(" [%s]: ", m.Method))
	lg.CW(w, lg.NCyan, fmt.Sprintf("%s", m.Path))
	lg.CW(w, lg.NWhite, fmt.Sprintf(" from:%s", m.ClientIP))
	if m.StatusCode >= 500 {
		lg.CW(w, lg.NRed, fmt.Sprintf(" - %s", http.StatusText(m.StatusCode)))
	} else if m.StatusCode >= 400 {
		lg.CW(w, lg.NYellow, fmt.Sprintf(" - %s", http.StatusText(m.StatusCode)))
	} else if m.StatusCode >= 300 {
		lg.CW(w, lg.NCyan, fmt.Sprintf(" - %s", http.StatusText(m.StatusCode)))
	} else {
		lg.CW(w, lg.NGreen, fmt.Sprintf(" - %s", http.StatusText(m.StatusCode)))
	}
	size := float64(m.ResponseSize) / 1024
	lg.CW(w, lg.NCyan, fmt.Sprintf(` Size:%.2f%s`, size, "KB"))
	lg.CW(w, lg.NWhite, " in ")
	limitLatency := time.Duration(envar.GetInt64(1000, "LIMIT_LATENCY")) * time.Millisecond
	if m.Latency < limitLatency {
		lg.CW(w, lg.NGreen, " Latency:%s", m.Latency)
	} else if m.Latency < 5*time.Second {
		lg.CW(w, lg.NYellow, " Latency:%s", m.Latency)
	} else {
		lg.CW(w, lg.NRed, " Latency:%s", m.Latency)
	}
	lg.CW(w, lg.NWhite, " Response:%s", m.ResponseTime)
	m.metrics = m.CallMetrics()
	if m.metrics.RequestsPerSecond > m.metrics.RequestsLimit {
		lg.CW(w, lg.NRed, " - Request:S:%vM:%vH:%vD:%vL:%v", m.metrics.RequestsPerSecond, m.metrics.RequestsPerMinute, m.metrics.RequestsPerHour, m.metrics.RequestsPerDay, m.metrics.RequestsLimit)
	} else if m.metrics.RequestsPerSecond > int(float64(m.metrics.RequestsLimit)*0.6) {
		lg.CW(w, lg.NYellow, " - Request:S:%vM:%vH:%vD:%vL:%v", m.metrics.RequestsPerSecond, m.metrics.RequestsPerMinute, m.metrics.RequestsPerHour, m.metrics.RequestsPerDay, m.metrics.RequestsLimit)
	} else {
		lg.CW(w, lg.NGreen, " - Request:S:%vM:%vH:%vD:%vL:%v", m.metrics.RequestsPerSecond, m.metrics.RequestsPerMinute, m.metrics.RequestsPerHour, m.metrics.RequestsPerDay, m.metrics.RequestsLimit)
	}
	lg.Println(w)

	return m.ToJson()
}

/**
* telemetry
* @return et.Json
**/
func (m *Metrics) telemetry() et.Json {
	result := m.ToJson()
	result["metric"] = m.metrics.ToJson()

	go event.Telemetry(et.Json{
		"response": m,
		"metric":   m.metrics.ToJson(),
	})

	if m.metrics.RequestsPerSecond > m.metrics.RequestsLimit {
		go event.Overflow(et.Json{
			"response": m,
			"metric":   m.metrics.ToJson(),
		})
	}

	return result
}

/**
* DoneFn
* @params rw *ResponseWriterWrapper
* @params r *http.Request
* @return et.Json
**/
func (m *Metrics) DoneFn(rw *ResponseWriterWrapper) et.Json {
	m.StatusCode = rw.StatusCode
	m.ResponseSize = rw.Size
	m.CallResponseTime()
	m.CallLatency()
	m.println()

	return m.telemetry()
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

	return m.println()
}

/**
* DoneRpc
* @params r et.Json
* @return et.Json
**/
func (m *Metrics) DoneRpc(r any) et.Json {
	str, ok := r.(string)
	if !ok {
		m.ResponseSize = 0
	} else {
		m.ResponseSize = len(str)
	}
	m.StatusCode = http.StatusOK
	m.CallResponseTime()
	m.CallLatency()
	m.println()

	return m.telemetry()
}

/**
* WriteResponse
* @params w http.ResponseWriter
* @params r *http.Request
* @params statusCode int
* @params e []byte
**/
func (m *Metrics) WriteResponse(w http.ResponseWriter, r *http.Request, statusCode int, e []byte) error {
	rw := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: statusCode}

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(statusCode)
	rw.Write(e)

	m.DoneFn(rw)
	return nil
}

/**
* JSON
* @params w http.ResponseWriter
* @params r *http.Request
* @params statusCode int
* @params dt interface{}
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
* @params w http.ResponseWriter
* @params r *http.Request
* @params statusCode int
* @params dt et.Item
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
* @params w http.ResponseWriter
* @params r *http.Request
* @params statusCode int
* @params dt et.Items
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
* @params w http.ResponseWriter
* @params r *http.Request
* @params statusCode int
* @params message string
**/
func (m *Metrics) HTTPError(w http.ResponseWriter, r *http.Request, statusCode int, message string) error {
	msg := et.Json{
		"message": message,
	}

	return m.JSON(w, r, statusCode, msg)
}

/**
* Unauthorized
* @params w http.ResponseWriter
* @params r *http.Request
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
