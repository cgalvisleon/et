package middleware

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/response"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
	"github.com/shirou/gopsutil/v3/mem"
)

type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
	Size       int
	Host       string
}

/**
* WriteHeader
* @params statusCode int
**/
func (rw *ResponseWriterWrapper) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
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

type Metrics struct {
	ReqID            string
	TimeBegin        time.Time
	TimeEnd          time.Time
	TimeExec         time.Time
	SearchTime       time.Duration
	ResponseTime     time.Duration
	Downtime         time.Duration
	Latency          time.Duration
	StatusCode       int
	Status           string
	ContentLength    int64
	Header           http.Header
	Host             string
	EndPoint         string
	Method           string
	RemoteAddr       string
	HostName         string
	Proto            string
	MTotal           uint64
	MUsed            uint64
	MFree            uint64
	PFree            float64
	RequestsHost     Request
	RequestsEndpoint Request
	Scheme           string
}

/**
* NewMetric
* @params r *http.Request
* @return *Metrics
**/
func NewMetric(r *http.Request) *Metrics {
	result := &Metrics{}
	result.TimeBegin = time.Now()
	result.ReqID = utility.NewId()
	result.EndPoint = r.URL.Path
	result.Method = r.Method
	result.Proto = r.Proto
	result.RemoteAddr = r.Header.Get("X-Forwarded-For")
	if result.RemoteAddr == "" {
		result.RemoteAddr = r.Header.Get("X-Real-IP")
	}
	if result.RemoteAddr == "" {
		result.RemoteAddr = r.RemoteAddr
	} else {
		result.RemoteAddr = strs.Split(result.RemoteAddr, ",")[0]
	}
	result.HostName, _ = os.Hostname()
	memory, err := mem.VirtualMemory()
	if err != nil {
		result.MFree = 0
		result.MTotal = 0
		result.MUsed = 0
		result.PFree = 0
	} else {
		result.MTotal = memory.Total
		result.MUsed = memory.Used
		result.MFree = memory.Total - memory.Used
		result.PFree = float64(result.MFree) / float64(result.MTotal) * 100
	}
	result.RequestsHost = callRequests(result.HostName)
	result.RequestsEndpoint = callRequests(result.EndPoint)
	result.Scheme = "http"
	if r.TLS != nil {
		result.Scheme = "https"
	}

	return result
}

/**
* println
* @return js.Json
**/
func (m *Metrics) println() js.Json {
	w := logs.Color(logs.NMagenta, fmt.Sprintf(" [%s]: ", m.Method))
	logs.CW(w, logs.NCyan, fmt.Sprintf("%s %s", m.EndPoint, m.Proto))
	logs.CW(w, logs.NWhite, fmt.Sprintf(" from %s", m.RemoteAddr))
	if m.StatusCode >= 500 {
		logs.CW(w, logs.NRed, fmt.Sprintf(" - %s", m.Status))
	} else if m.StatusCode >= 400 {
		logs.CW(w, logs.NYellow, fmt.Sprintf(" - %s", m.Status))
	} else if m.StatusCode >= 300 {
		logs.CW(w, logs.NCyan, fmt.Sprintf(" - %s", m.Status))
	} else {
		logs.CW(w, logs.NGreen, fmt.Sprintf(" - %s", m.Status))
	}
	if m.ContentLength > 0 {
		logs.CW(w, logs.NCyan, fmt.Sprintf(" %v%s", m.ContentLength, "KB"))
	}
	logs.CW(w, logs.NWhite, " in ")
	if m.Latency < 500*time.Millisecond {
		logs.CW(w, logs.NGreen, "Latency:%s", m.Latency)
	} else if m.Latency < 5*time.Second {
		logs.CW(w, logs.NYellow, "Latency:%s", m.Latency)
	} else {
		logs.CW(w, logs.NRed, "Latency:%s", m.Latency)
	}
	logs.CW(w, logs.NWhite, " Response:%s", m.ResponseTime)
	logs.CW(w, logs.NRed, " Downtime:%s", m.Downtime)
	if m.RequestsHost.Seccond > m.RequestsHost.Limit {
		logs.CW(w, logs.NRed, " - Request S:%vM:%vH:%vL:%v", m.RequestsHost.Seccond, m.RequestsHost.Minute, m.RequestsHost.Hour, m.RequestsHost.Limit)
	} else {
		logs.CW(w, logs.NYellow, " - Request S:%vM:%vH:%vL:%v", m.RequestsHost.Seccond, m.RequestsHost.Minute, m.RequestsHost.Hour, m.RequestsHost.Limit)
	}
	logs.Println(w)

	result := js.Json{
		"reqID":         m.ReqID,
		"time_begin":    m.TimeBegin,
		"time_end":      m.TimeEnd,
		"time_exec":     m.TimeExec,
		"latency":       m.Latency,
		"search_time":   m.SearchTime,
		"response_time": m.ResponseTime,
		"host_name":     m.HostName,
		"remote_addr":   m.RemoteAddr,
		"request": js.Json{
			"end_point": m.EndPoint,
			"method":    m.Method,
			"status":    m.Status,
			"bytes":     m.ContentLength,
			"header":    m.Header,
			"scheme":    m.Scheme,
			"host":      m.Host,
		},
		"memory": js.Json{
			"unity":        "MB",
			"total":        m.MTotal / 1024 / 1024,
			"used":         m.MUsed / 1024 / 1024,
			"free":         m.MFree / 1024 / 1024,
			"percent_free": math.Floor(m.PFree*100) / 100,
		},
		"request_host": js.Json{
			"host":   m.RequestsHost.Tag,
			"day":    m.RequestsHost.Day,
			"hour":   m.RequestsHost.Hour,
			"minute": m.RequestsHost.Minute,
			"second": m.RequestsHost.Seccond,
			"limit":  m.RequestsHost.Limit,
		},
		"requests_endpoint": js.Json{
			"endpoint": m.RequestsEndpoint.Tag,
			"day":      m.RequestsEndpoint.Day,
			"hour":     m.RequestsEndpoint.Hour,
			"minute":   m.RequestsEndpoint.Minute,
			"second":   m.RequestsEndpoint.Seccond,
			"limit":    m.RequestsEndpoint.Limit,
		},
	}

	go event.Telemetry(result)

	if m.RequestsHost.Seccond > m.RequestsHost.Limit {
		go event.Overflow(result)
	}

	return result
}

/**
* CallExecute
**/
func (m *Metrics) CallExecute() {
	m.SearchTime = time.Since(m.TimeBegin)
	m.TimeExec = time.Now()
}

/**
* Done
* @params res *http.Response
* @return js.Json
**/
func (m *Metrics) Done(res *http.Response) js.Json {
	m.TimeEnd = time.Now()
	m.ResponseTime = time.Since(m.TimeExec)
	m.Latency = time.Since(m.TimeBegin)
	m.Downtime = m.SearchTime - m.ResponseTime
	m.StatusCode = res.StatusCode
	m.Status = res.Status
	m.ContentLength = res.ContentLength
	m.Header = res.Header
	m.Host = res.Request.Host

	return m.println()
}

/**
* DoneFn
* @params statusCode int
* @params w http.ResponseWriter
* @params r *http.Request
* @return js.Json
**/
func (m *Metrics) DoneFn(statusCode int, w http.ResponseWriter, r *http.Request) js.Json {
	rw := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: statusCode, Host: r.Host}
	m.TimeEnd = time.Now()
	m.ResponseTime = time.Since(m.TimeExec)
	m.Latency = time.Since(m.TimeBegin)
	m.Downtime = m.Latency - m.ResponseTime
	m.StatusCode = rw.StatusCode
	m.Status = http.StatusText(rw.StatusCode)
	m.ContentLength = int64(rw.Size)
	m.Header = rw.Header()
	m.Host = rw.Host

	return m.println()
}

/**
* Unauthorized
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (m *Metrics) Unauthorized(w http.ResponseWriter, r *http.Request) {
	m.CallExecute()
	response.HTTPError(w, r, http.StatusUnauthorized, "401 Unauthorized")
	go m.DoneFn(http.StatusUnauthorized, w, r)
}

/**
* NotFound
* @params handler http.HandlerFunc
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (m *Metrics) NotFound(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	m.CallExecute()
	handler(w, r)
	go m.DoneFn(http.StatusNotFound, w, r)
}

/**
* Handler
* @params handler http.HandlerFunc
* @params w http.ResponseWriter
* @params r *http.Request
**/
func (m *Metrics) Handler(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	handler(w, r)
	go m.DoneFn(http.StatusOK, w, r)
}
