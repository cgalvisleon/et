package middleware

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	lg "github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var hostName, _ = os.Hostname()

type Result struct {
	Ok     bool        `json:"ok"`
	Result interface{} `json:"result"`
}

type ContentLength struct {
	Header int
	Body   int
	Total  int
}

type ResponseWriterWrapper struct {
	http.ResponseWriter
	StatusCode int
	SizeHeader int
	SizeBody   int
	SizeTotal  int
	Host       string
}

/**
* WriteHeader
* @params statusCode int
**/
func (rw *ResponseWriterWrapper) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
	totalHeader := 0
	for key, values := range rw.Header() {
		totalHeader += len(key)
		for _, value := range values {
			totalHeader += len(value) + len(": ") + len("\r\n")
		}
	}
	totalHeader += len("\r\n") * len(rw.Header())
	rw.SizeHeader = totalHeader
	rw.SizeTotal = rw.SizeHeader + rw.SizeBody
}

/**
* Write
* @params b []byte
**/
func (rw *ResponseWriterWrapper) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.SizeBody += size
	rw.SizeTotal = rw.SizeHeader + rw.SizeBody
	return size, err
}

/**
* ContentLength
* @return ContentLength
**/
func (rw *ResponseWriterWrapper) ContentLength() ContentLength {
	totalHeader := 0
	for key, values := range rw.Header() {
		totalHeader += len(key)
		for _, value := range values {
			totalHeader += len(value) + len(": ") + len("\r\n")
		}
	}
	totalHeader += len("\r\n") * len(rw.Header())
	rw.SizeHeader = totalHeader
	rw.SizeTotal = rw.SizeHeader + rw.SizeBody
	return ContentLength{
		Header: rw.SizeHeader,
		Body:   rw.SizeBody,
		Total:  rw.SizeTotal,
	}
}

/**
* headerLength
* @params res *http.Response
* @return int
**/
func headerLength(res *http.Response) int {
	result := 0
	for key, values := range res.Header {
		result += len(key)
		for _, value := range values {
			result += len(value) + len(": ") + len("\r\n")
		}
	}
	result += len("\r\n") * len(res.Header)

	return result
}

/**
* contentLength
* @params res *http.Response
* @return int
**/
func contentLength(res *http.Response) ContentLength {
	result := headerLength(res)

	return ContentLength{
		Header: result,
		Body:   int(res.ContentLength),
		Total:  result + int(res.ContentLength),
	}
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
	ContentLength    ContentLength
	Header           http.Header
	Host             string
	EndPoint         string
	Method           string
	Proto            string
	RemoteAddr       string
	HostName         string
	RequestsHost     Request
	RequestsEndpoint Request
	Scheme           string
	CPUUsage         float64
	MemoryTotal      uint64
	MemoeryUsage     uint64
	MmemoryFree      uint64
}

/**
* NewMetric
* @params r *http.Request
**/
func NewMetric(r *http.Request) *Metrics {
	remoteAddr := r.Header.Get("X-Forwarded-For")
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
	host := envar.GetStr(hostName, "HOST")
	endPoint := strs.Format(`%s/%s`, host, r.URL.Path)

	result := &Metrics{
		TimeBegin:        timezone.NowTime(),
		ReqID:            utility.UUID(),
		EndPoint:         r.URL.Path,
		Method:           r.Method,
		Proto:            r.Proto,
		RemoteAddr:       remoteAddr,
		Host:             host,
		HostName:         hostName,
		RequestsHost:     callRequests(hostName),
		RequestsEndpoint: callRequests(endPoint),
		Scheme:           scheme,
	}

	go result.CallCPUUsage()
	go result.CallMemoryUsage()

	return result
}

/**
* NewRpcMetric
* @params r *http.Request
**/
func NewRpcMetric(method string) *Metrics {
	endPoint := method
	scheme := "rpc"

	result := &Metrics{
		TimeBegin:        timezone.NowTime(),
		ReqID:            utility.UUID(),
		EndPoint:         endPoint,
		Method:           "RPC",
		Proto:            "Http/1.2",
		HostName:         hostName,
		RequestsHost:     callRequests(hostName),
		RequestsEndpoint: callRequests(endPoint),
		Scheme:           scheme,
	}

	go result.CallCPUUsage()
	go result.CallMemoryUsage()

	return result
}

/**
* SetAddress
* @params address string
* @return *Metrics
**/
func (m *Metrics) SetAddress(address string) *Metrics {
	m.RemoteAddr = address

	return m
}

/**
* CallCPUUsage
**/
func (m *Metrics) CallCPUUsage() {
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		m.CPUUsage = 0
	}
	m.CPUUsage = percentages[0]
}

/**
* CallMemoryUsage
**/
func (m *Metrics) CallMemoryUsage() {
	v, err := mem.VirtualMemory()
	if err != nil {
		m.MemoryTotal = 0
		m.MemoeryUsage = 0
		m.MmemoryFree = 0
	}
	m.MemoryTotal = v.Total
	m.MemoeryUsage = v.Used
	m.MmemoryFree = v.Free
}

/**
* CallSearchTime
**/
func (m *Metrics) CallSearchTime() {
	m.SearchTime = time.Since(m.TimeBegin)
	m.TimeExec = timezone.NowTime()
}

/**
* DoneFn
* @params rw *ResponseWriterWrapper
* @params r *http.Request
* @return et.Json
**/
func (m *Metrics) DoneFn(rw *ResponseWriterWrapper) et.Json {
	m.TimeEnd = timezone.NowTime()
	m.ResponseTime = time.Since(m.TimeExec)
	m.Latency = time.Since(m.TimeBegin)
	m.Downtime = m.Latency - m.ResponseTime
	m.StatusCode = rw.StatusCode
	m.Status = http.StatusText(m.StatusCode)
	m.ContentLength = ContentLength{
		Header: rw.SizeHeader,
		Body:   rw.SizeBody,
		Total:  rw.SizeTotal,
	}
	m.Header = rw.Header()

	return m.println()
}

func (m *Metrics) DoneRpc(r interface{}) et.Json {
	var size int
	jsonData, err := json.Marshal(r)
	if err != nil {
		size = 0
	} else {
		size = len(jsonData)
		size = size / 1024
	}

	m.TimeEnd = timezone.NowTime()
	m.ResponseTime = time.Since(m.TimeExec)
	m.Latency = time.Since(m.TimeBegin)
	m.Downtime = m.Latency - m.ResponseTime
	m.StatusCode = http.StatusOK
	m.Status = http.StatusText(m.StatusCode)
	m.ContentLength = ContentLength{
		Header: 0,
		Body:   size,
		Total:  size,
	}

	return m.println()
}

func (m *Metrics) WriteResponse(w http.ResponseWriter, r *http.Request, statusCode int, e []byte) error {
	rw := &ResponseWriterWrapper{ResponseWriter: w, StatusCode: statusCode, Host: r.Host}

	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(statusCode)
	rw.Write(e)

	m.DoneFn(rw)
	return nil
}

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

	if !dt.Ok {
		statusCode = http.StatusNotFound
	}

	return m.WriteResponse(w, r, statusCode, e)
}

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

	if !dt.Ok {
		statusCode = http.StatusNotFound
	}

	return m.WriteResponse(w, r, statusCode, e)
}

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

/**
* println
* @return et.Json
**/
func (m *Metrics) println() et.Json {
	w := lg.Color(lg.NMagenta, fmt.Sprintf(" [%s]: ", m.Method))
	lg.CW(w, lg.NCyan, fmt.Sprintf("%s %s", m.EndPoint, m.Proto))
	lg.CW(w, lg.NWhite, fmt.Sprintf(" from %s", m.RemoteAddr))
	if m.StatusCode >= 500 {
		lg.CW(w, lg.NRed, fmt.Sprintf(" - %s", m.Status))
	} else if m.StatusCode >= 400 {
		lg.CW(w, lg.NYellow, fmt.Sprintf(" - %s", m.Status))
	} else if m.StatusCode >= 300 {
		lg.CW(w, lg.NCyan, fmt.Sprintf(" - %s", m.Status))
	} else {
		lg.CW(w, lg.NGreen, fmt.Sprintf(" - %s", m.Status))
	}
	lg.CW(w, lg.NCyan, fmt.Sprintf(" Size: %v%s", m.ContentLength.Total, "KB"))
	lg.CW(w, lg.NWhite, " in ")
	limitLatency := time.Duration(envar.GetInt64(500, "LIMIT_LATENCY")) * time.Millisecond
	if m.Latency < limitLatency {
		lg.CW(w, lg.NGreen, "Latency:%s", m.Latency)
	} else if m.Latency < 5*time.Second {
		lg.CW(w, lg.NYellow, "Latency:%s", m.Latency)
	} else {
		lg.CW(w, lg.NRed, "Latency:%s", m.Latency)
	}
	lg.CW(w, lg.NWhite, " Response:%s", m.ResponseTime)
	lg.CW(w, lg.NRed, " Downtime:%s", m.Downtime)
	if m.RequestsHost.Seccond > m.RequestsHost.Limit {
		lg.CW(w, lg.NRed, " - Request S:%vM:%vH:%vD:%vL:%v", m.RequestsHost.Seccond, m.RequestsHost.Minute, m.RequestsHost.Hour, m.RequestsHost.Day, m.RequestsHost.Limit)
	} else {
		lg.CW(w, lg.NYellow, " - Request S:%vM:%vH:%vD:%vL:%v", m.RequestsHost.Seccond, m.RequestsHost.Minute, m.RequestsHost.Hour, m.RequestsHost.Day, m.RequestsHost.Limit)
	}
	lg.Println(w)

	result := et.Json{
		"reqID":         m.ReqID,
		"time_begin":    m.TimeBegin,
		"time_end":      m.TimeEnd,
		"time_exec":     m.TimeExec,
		"latency":       m.Latency,
		"search_time":   m.SearchTime,
		"response_time": m.ResponseTime,
		"host_name":     m.HostName,
		"remote_addr":   m.RemoteAddr,
		"request": et.Json{
			"end_point": m.EndPoint,
			"method":    m.Method,
			"status":    m.Status,
			"size": et.Json{
				"header": m.ContentLength.Header,
				"body":   m.ContentLength.Body,
			},
			"header": m.Header,
			"scheme": m.Scheme,
			"host":   m.Host,
		},
		"system": et.Json{
			"unity":        "MB",
			"total":        m.MemoryTotal / 1024 / 1024,
			"used":         m.MemoeryUsage / 1024 / 1024,
			"free":         m.MmemoryFree / 1024 / 1024,
			"percent_free": math.Floor(float64(m.MmemoryFree) / float64(m.MemoryTotal)),
			"cpu_usage":    m.CPUUsage,
		},
		"request_host": et.Json{
			"host":   m.RequestsHost.Tag,
			"day":    m.RequestsHost.Day,
			"hour":   m.RequestsHost.Hour,
			"minute": m.RequestsHost.Minute,
			"second": m.RequestsHost.Seccond,
			"limit":  m.RequestsHost.Limit,
		},
		"requests_endpoint": et.Json{
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
