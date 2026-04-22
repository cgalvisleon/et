package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/request"
	lg "github.com/cgalvisleon/et/stdrout"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

var (
	hostName, _ = os.Hostname()
)

const (
	TELEMETRY                                   = "telemetry"
	TELEMETRY_LOG                               = "telemetry:log"
	TELEMETRY_OVERFLOW                          = "telemetry:overflow"
	TELEMETRY_TOKEN_LAST_USE                    = "telemetry:token:last_use"
	MetricKey                request.ContextKey = "metric"
)

// ResponseWriterWrapper wraps http.ResponseWriter to capture status code and response size.
type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
	Size       int
}

// WriteHeader intercepts the status code before delegating to the underlying writer.
func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write delegates to the underlying writer and accumulates the bytes written.
func (rw *ResponseWriterWrapper) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.Size += size
	return size, err
}

// Metrics holds observability data for a single HTTP or RPC request.
// Field names and semantics follow OpenTelemetry semantic conventions for HTTP:
// https://opentelemetry.io/docs/specs/semconv/http/
type Metrics struct {
	// timestamp — request start time (RFC3339)
	TimeStamp time.Time `json:"timestamp"`
	// Internal correlation ID propagated via ServiceId header
	ServiceId string `json:"service_id"`
	// client.address — originating client IP (X-Forwarded-For > X-Real-IP > RemoteAddr)
	ClientAddress string `json:"client_address"`
	// url.scheme — "http" or "https"
	Scheme string `json:"scheme"`
	// server.address — hostname of this server
	ServerAddress string `json:"server_address"`
	// http.request.method
	Method string `json:"method"`
	// url.path
	Path string `json:"path"`
	// url.query — raw query string
	Query string `json:"query"`
	// http.response.status_code
	StatusCode int `json:"status_code"`
	// http.request.body.size — bytes received
	RequestSize int `json:"request_size"`
	// http.response.body.size — bytes sent
	ResponseSize int `json:"response_size"`
	// user_agent.original
	UserAgent string `json:"user_agent"`
	// trace_id — from X-Trace-ID or X-Request-ID header
	TraceID string `json:"trace_id"`
	// app_name — client application identifier (AppName header)
	AppName string `json:"app_name"`
	// Search/query phase duration in milliseconds
	SearchTime float64 `json:"search_time_ms"`
	// Handler processing duration in milliseconds
	ResponseTime float64 `json:"response_time_ms"`
	// http.server.request.duration — total request latency in milliseconds
	Latency float64 `json:"latency_ms"`

	key     string
	mark    time.Time
	metrics Telemetry
}

// ToJson returns a JSON-serializable representation of the metrics.
func (s *Metrics) ToJson() et.Json {
	return et.Json{
		"timestamp":        s.TimeStamp.Format(time.RFC3339),
		"service_id":       s.ServiceId,
		"client_address":   s.ClientAddress,
		"scheme":           s.Scheme,
		"server_address":   s.ServerAddress,
		"method":           s.Method,
		"path":             s.Path,
		"query":            s.Query,
		"status_code":      s.StatusCode,
		"request_size":     s.RequestSize,
		"response_size":    s.ResponseSize,
		"user_agent":       s.UserAgent,
		"trace_id":         s.TraceID,
		"app_name":         s.AppName,
		"search_time_ms":   s.SearchTime,
		"response_time_ms": s.ResponseTime,
		"latency_ms":       s.Latency,
	}
}

// Telemetry holds rate-based request counters for a specific endpoint key.
type Telemetry struct {
	TimeStamp         string `json:"timestamp"`
	AppName           string `json:"service_name"`
	Key               string `json:"key"`
	RequestsPerSecond int64  `json:"requests_per_second"`
	RequestsPerMinute int64  `json:"requests_per_minute"`
	RequestsPerHour   int64  `json:"requests_per_hour"`
	RequestsPerDay    int64  `json:"requests_per_day"`
	RequestsLimit     int64  `json:"requests_limit"`
}

// ToJson returns a JSON-serializable representation of the telemetry counters.
func (t *Telemetry) ToJson() et.Json {
	return et.Json{
		"timestamp":           t.TimeStamp,
		"app_name":            t.AppName,
		"key":                 t.Key,
		"requests_per_second": t.RequestsPerSecond,
		"requests_per_minute": t.RequestsPerMinute,
		"requests_per_hour":   t.RequestsPerHour,
		"requests_per_day":    t.RequestsPerDay,
		"requests_limit":      t.RequestsLimit,
	}
}

// GetMetrics retrieves the Metrics stored in the request context, or creates a new one.
func GetMetrics(r *http.Request) *Metrics {
	metric, ok := r.Context().Value(MetricKey).(*Metrics)
	if !ok {
		return NewMetric(r)
	}
	return metric
}

// PushTelemetry publishes a telemetry event asynchronously.
func PushTelemetry(data et.Json) {
	go event.Publish(TELEMETRY, data)
}

// PushTelemetryLog publishes a log line as a telemetry event asynchronously.
func PushTelemetryLog(data string) {
	go event.Publish(TELEMETRY_LOG, et.Json{"log": data})
}

// PushTelemetryOverflow publishes a rate-limit overflow event asynchronously.
func PushTelemetryOverflow(data et.Json) {
	go event.Publish(TELEMETRY_OVERFLOW, data)
}

// PushTokenLastUse publishes a token last-use event asynchronously.
func PushTokenLastUse(data et.Json) {
	go event.Publish(TELEMETRY_TOKEN_LAST_USE, data)
}

// NewMetric creates a Metrics instance populated from the incoming HTTP request.
// Client IP resolution order: X-Forwarded-For → X-Real-IP → RemoteAddr.
func NewMetric(r *http.Request) *Metrics {
	clientAddr := r.Header.Get("X-Forwarded-For")
	if clientAddr == "" {
		clientAddr = r.Header.Get("X-Real-IP")
	}
	if clientAddr == "" {
		clientAddr = r.RemoteAddr
	}
	clientAddr = strs.Split(clientAddr, ",")[0]

	serviceId := r.Header.Get("ServiceId")
	if serviceId == "" {
		serviceId = reg.ULID()
		r.Header.Set("ServiceId", serviceId)
	}

	traceID := r.Header.Get("X-Trace-ID")
	if traceID == "" {
		traceID = r.Header.Get("X-Request-ID")
	}

	appName := r.Header.Get("AppName")
	if appName == "" {
		appName = "Not Found"
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	now := timezone.Now()
	return &Metrics{
		TimeStamp:     now,
		ServiceId:     serviceId,
		ClientAddress: clientAddr,
		Scheme:        scheme,
		ServerAddress: hostName,
		Method:        r.Method,
		Path:          r.URL.Path,
		Query:         r.URL.RawQuery,
		UserAgent:     r.UserAgent(),
		TraceID:       traceID,
		AppName:       appName,
		RequestSize:   int(r.ContentLength),
		mark:          now,
		key:           fmt.Sprintf(`%s:%s`, r.Method, r.URL.Path),
	}
}

// NewRpcMetric creates a Metrics instance for a JSON-RPC call.
func NewRpcMetric(method string) *Metrics {
	now := timezone.Now()
	return &Metrics{
		TimeStamp: now,
		ServiceId: utility.UUID(),
		Scheme:    "rpc",
		Method:    "RPC",
		Path:      method,
		mark:      now,
		key:       fmt.Sprintf(`RPC:%s`, method),
	}
}

// setRequest adds or removes the request key from the in-flight tracking list.
func (s *Metrics) setRequest(remove bool) {
	s.key = fmt.Sprintf(`%s:%s`, s.Method, s.Path)
	if remove {
		cache.LRem("telemetry:requests", s.key)
	} else {
		cache.LPush("telemetry:requests", s.key)
	}
}

// SetPath updates the matched route path and registers the request key.
func (s *Metrics) SetPath(val string) {
	if val == "" {
		return
	}
	s.Path = val
	s.setRequest(false)
}

// CallSearchTime records the duration of the search/query phase in milliseconds.
func (s *Metrics) CallSearchTime() {
	s.SearchTime = float64(time.Since(s.mark).Milliseconds())
	s.mark = timezone.Now()
}

// CallResponseTime records the handler processing duration in milliseconds.
func (s *Metrics) CallResponseTime() {
	s.ResponseTime = float64(time.Since(s.mark).Milliseconds())
	s.mark = timezone.Now()
}

// CallLatency records the total end-to-end latency in milliseconds.
func (s *Metrics) CallLatency() {
	s.Latency = float64(time.Since(s.TimeStamp).Milliseconds())
}

// CallMetrics computes rolling request-rate counters for the current endpoint key.
func (s *Metrics) CallMetrics() Telemetry {
	timeNow := timezone.Now()
	date := timeNow.Format("2006-01-02")
	hour := timeNow.Format("2006-01-02-15")
	minute := timeNow.Format("2006-01-02-15:04")
	second := timeNow.Format("2006-01-02-15:04:05")
	requestsLimit := envar.GetInt("REQUESTS_LIMIT", 400)

	return Telemetry{
		TimeStamp:         date,
		Key:               s.key,
		RequestsPerSecond: cache.Incr(reg.GenHashKey(s.key, second), 2),
		RequestsPerMinute: cache.Incr(reg.GenHashKey(s.key, minute), 60),
		RequestsPerHour:   cache.Incr(reg.GenHashKey(s.key, hour), 3600),
		RequestsPerDay:    cache.Incr(reg.GenHashKey(s.key, date), 86400),
		RequestsLimit:     int64(requestsLimit),
	}
}

// logRequest prints a color-coded summary line to stdout and publishes the log event.
func (s *Metrics) logRequest() et.Json {
	w := lg.Color(nil, lg.Reset, "%s", timezone.NowStr())
	lg.Color(w, lg.Purple, " [%s]: ", s.Method)
	lg.Color(w, lg.Cyan, "%s", s.Path)
	if s.Query != "" {
		lg.Color(w, lg.White, "?%s", s.Query)
	}
	lg.Color(w, lg.White, " from:%s", s.ClientAddress)

	switch {
	case s.StatusCode >= 500:
		lg.Color(w, lg.Red, " - %s", http.StatusText(s.StatusCode))
	case s.StatusCode >= 400:
		lg.Color(w, lg.Yellow, " - %s", http.StatusText(s.StatusCode))
	case s.StatusCode >= 300:
		lg.Color(w, lg.Cyan, " - %s", http.StatusText(s.StatusCode))
	default:
		lg.Color(w, lg.Green, " - %s", http.StatusText(s.StatusCode))
	}

	size := float64(s.ResponseSize) / 1024
	lg.Color(w, lg.Cyan, " Size:%.2fKB", size)

	limitLatency := time.Duration(envar.GetInt("LATENCY_LIMIT", 1000)) * time.Millisecond
	latencyDur := time.Duration(s.Latency * float64(time.Millisecond))
	switch {
	case latencyDur >= 5*time.Second:
		lg.Color(w, lg.Red, " Latency:%.2fms", s.Latency)
	case latencyDur >= limitLatency:
		lg.Color(w, lg.Yellow, " Latency:%.2fms", s.Latency)
	default:
		lg.Color(w, lg.Green, " Latency:%.2fms", s.Latency)
	}

	lg.Color(w, lg.White, " Response:%.2fms", s.ResponseTime)

	s.metrics = s.CallMetrics()
	threshold := float64(s.metrics.RequestsLimit) * 0.6
	rps := s.metrics.RequestsPerSecond
	switch {
	case rps > s.metrics.RequestsLimit:
		lg.Color(w, lg.Red, " Req S:%v M:%v H:%v D:%v L:%v",
			rps, s.metrics.RequestsPerMinute, s.metrics.RequestsPerHour, s.metrics.RequestsPerDay, s.metrics.RequestsLimit)
	case rps > int64(threshold):
		lg.Color(w, lg.Yellow, " Req S:%v M:%v H:%v D:%v L:%v",
			rps, s.metrics.RequestsPerMinute, s.metrics.RequestsPerHour, s.metrics.RequestsPerDay, s.metrics.RequestsLimit)
	default:
		lg.Color(w, lg.Green, " Req S:%v M:%v H:%v D:%v L:%v",
			rps, s.metrics.RequestsPerMinute, s.metrics.RequestsPerHour, s.metrics.RequestsPerDay, s.metrics.RequestsLimit)
	}

	lg.Color(w, lg.Cyan, " [ServiceId]:%s", s.ServiceId)
	if s.TraceID != "" {
		lg.Color(w, lg.Cyan, " [TraceId]:%s", s.TraceID)
	}
	lg.Color(w, lg.White, " [App]:%s", s.AppName)
	println(*w)

	s.setRequest(true)
	PushTelemetryLog(*w)

	return s.ToJson()
}

// telemetry emits the final telemetry events and returns the combined result.
func (s *Metrics) telemetry() et.Json {
	result := s.ToJson()
	result["metric"] = s.metrics.ToJson()

	payload := et.Json{
		"response": s.ToJson(),
		"metric":   s.metrics.ToJson(),
	}
	PushTelemetry(payload)

	if s.metrics.RequestsPerSecond > s.metrics.RequestsLimit {
		PushTelemetryOverflow(payload)
	}

	return result
}

// DoneHTTP finalizes metrics after an HTTP response has been written.
func (s *Metrics) DoneHTTP(rw *ResponseWriterWrapper) et.Json {
	s.StatusCode = rw.StatusCode
	s.ResponseSize = rw.Size
	s.CallResponseTime()
	s.CallLatency()
	s.logRequest()
	return s.telemetry()
}

// DoneRpc finalizes metrics after an RPC call has returned.
func (s *Metrics) DoneRpc(r any) et.Json {
	switch v := r.(type) {
	case string:
		s.ResponseSize = len(v)
	case et.Json:
		s.ResponseSize = len(v.ToString())
	case []byte:
		s.ResponseSize = len(v)
	case int:
		s.ResponseSize = len(strconv.Itoa(v))
	case float64:
		s.ResponseSize = len(strconv.FormatFloat(v, 'f', -1, 64))
	case bool:
		s.ResponseSize = len(strconv.FormatBool(v))
	case et.List:
		s.ResponseSize = len(v.ToString())
	case et.Items:
		s.ResponseSize = len(v.ToString())
	case et.Item:
		s.ResponseSize = len(v.ToString())
	default:
		s.ResponseSize = len(fmt.Sprintf("%v", v))
	}

	s.StatusCode = http.StatusOK
	s.CallResponseTime()
	s.CallLatency()
	s.logRequest()
	return s.telemetry()
}

// WriteResponse writes a raw JSON byte response, records metrics, and returns nil on success.
func (s *Metrics) WriteResponse(w http.ResponseWriter, r *http.Request, statusCode int, e []byte) error {
	rw := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: statusCode}
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(statusCode)
	rw.Write(e)
	s.DoneHTTP(rw)
	return nil
}

// RESULT serializes data as JSON and writes the response.
func (s *Metrics) RESULT(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) error {
	if data == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}
	e, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.WriteResponse(w, r, statusCode, e)
}

// JSON wraps data in a standard {ok, result} envelope and writes the response.
func (s *Metrics) JSON(w http.ResponseWriter, r *http.Request, statusCode int, dt interface{}) error {
	if dt == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}
	result := struct {
		Ok     bool        `json:"ok"`
		Result interface{} `json:"result"`
	}{
		Ok:     statusCode == http.StatusOK,
		Result: dt,
	}
	e, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return s.WriteResponse(w, r, statusCode, e)
}

// ITEM serializes an et.Item and writes the response.
func (s *Metrics) ITEM(w http.ResponseWriter, r *http.Request, statusCode int, dt et.Item) error {
	if &dt == (&et.Item{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}
	e, err := json.Marshal(dt)
	if err != nil {
		return err
	}
	return s.WriteResponse(w, r, statusCode, e)
}

// ITEMS serializes an et.Items and writes the response.
func (s *Metrics) ITEMS(w http.ResponseWriter, r *http.Request, statusCode int, dt et.Items) error {
	if &dt == (&et.Items{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		return nil
	}
	e, err := json.Marshal(dt)
	if err != nil {
		return err
	}
	return s.WriteResponse(w, r, statusCode, e)
}

// HTTPError writes a JSON error response with the given status code and message.
func (s *Metrics) HTTPError(w http.ResponseWriter, r *http.Request, statusCode int, message string) error {
	return s.JSON(w, r, statusCode, et.Json{"message": message})
}
